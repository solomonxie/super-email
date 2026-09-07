package config

import (
	"testing"
	"time"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"LLM_PROVIDER", "OLLAMA_HOST", "OLLAMA_MODEL",
		"OPENAI_API_KEY", "OPENAI_MODEL",
		"TEMPORAL_ADDRESS", "TEMPORAL_NAMESPACE", "TEMPORAL_TASK_QUEUE",
		"MAX_ITERATIONS", "ASK_USER_TIMEOUT",
	} {
		t.Setenv(k, "")
	}
}

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.LLMProvider != LLMProviderOllama {
		t.Errorf("LLMProvider = %q, want %q", cfg.LLMProvider, LLMProviderOllama)
	}
	if cfg.MaxIterations != 10 {
		t.Errorf("MaxIterations = %d, want 10", cfg.MaxIterations)
	}
	if cfg.AskUserTimeout != 72*time.Hour {
		t.Errorf("AskUserTimeout = %v, want 72h", cfg.AskUserTimeout)
	}
}

func TestLoadOpenAIRequiresKey(t *testing.T) {
	clearEnv(t)
	t.Setenv("LLM_PROVIDER", "openai")

	if _, err := Load(); err == nil {
		t.Fatal("Load() with LLM_PROVIDER=openai and no key: want error, got nil")
	}

	t.Setenv("OPENAI_API_KEY", "sk-test")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.LLMProvider != LLMProviderOpenAI {
		t.Errorf("LLMProvider = %q, want %q", cfg.LLMProvider, LLMProviderOpenAI)
	}
}

func TestLoadUnknownProvider(t *testing.T) {
	clearEnv(t)
	t.Setenv("LLM_PROVIDER", "bogus")

	if _, err := Load(); err == nil {
		t.Fatal("Load() with bogus LLM_PROVIDER: want error, got nil")
	}
}

func TestLoadBadMaxIterations(t *testing.T) {
	clearEnv(t)
	t.Setenv("MAX_ITERATIONS", "not-a-number")

	if _, err := Load(); err == nil {
		t.Fatal("Load() with bad MAX_ITERATIONS: want error, got nil")
	}
}
