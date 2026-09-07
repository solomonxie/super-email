// Package config loads super-email's runtime configuration from
// environment variables (.env locally, real env vars in deploy — see
// DESIGN.md §9).
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// LLMProvider selects the agent.Client backend.
type LLMProvider string

const (
	LLMProviderOllama LLMProvider = "ollama"
	LLMProviderOpenAI LLMProvider = "openai"
)

// Config is the full set of environment-derived settings.
type Config struct {
	LLMProvider LLMProvider

	OllamaHost  string
	OllamaModel string

	OpenAIAPIKey string
	OpenAIModel  string

	TemporalAddress   string
	TemporalNamespace string
	TemporalTaskQueue string

	MaxIterations  int
	AskUserTimeout time.Duration
}

// Load reads Config from the environment, applying defaults for anything
// unset. It returns an error only when a set value can't be parsed, or a
// required-for-the-selected-provider value is missing.
func Load() (Config, error) {
	cfg := Config{
		LLMProvider: LLMProvider(getEnv("LLM_PROVIDER", string(LLMProviderOllama))),

		OllamaHost:  getEnv("OLLAMA_HOST", "http://localhost:11434"),
		OllamaModel: getEnv("OLLAMA_MODEL", "llama3.1"),

		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:  getEnv("OPENAI_MODEL", "gpt-4o-mini"),

		TemporalAddress:   getEnv("TEMPORAL_ADDRESS", "localhost:7233"),
		TemporalNamespace: getEnv("TEMPORAL_NAMESPACE", "default"),
		TemporalTaskQueue: getEnv("TEMPORAL_TASK_QUEUE", "super-email"),
	}

	maxIter, err := getEnvInt("MAX_ITERATIONS", 10)
	if err != nil {
		return Config{}, err
	}
	cfg.MaxIterations = maxIter

	askTimeout, err := getEnvDuration("ASK_USER_TIMEOUT", 72*time.Hour)
	if err != nil {
		return Config{}, err
	}
	cfg.AskUserTimeout = askTimeout

	switch cfg.LLMProvider {
	case LLMProviderOllama:
		// no required secret
	case LLMProviderOpenAI:
		if cfg.OpenAIAPIKey == "" {
			return Config{}, fmt.Errorf("LLM_PROVIDER=openai requires OPENAI_API_KEY")
		}
	default:
		return Config{}, fmt.Errorf("unknown LLM_PROVIDER %q (want %q or %q)", cfg.LLMProvider, LLMProviderOllama, LLMProviderOpenAI)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: %w", key, v, err)
	}
	return n, nil
}

func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: %w", key, v, err)
	}
	return d, nil
}
