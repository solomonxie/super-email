// Command worker is the Temporal worker: it registers AgentLoopWorkflow
// and its activities on the task queue and polls for work. See
// DESIGN.md §2/§8.
package main

import (
	"log"
	"log/slog"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/solomonxie/super-email/internal/agent"
	"github.com/solomonxie/super-email/internal/config"
	sewf "github.com/solomonxie/super-email/internal/workflow"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	agentClient, err := agent.NewClient(cfg)
	if err != nil {
		log.Fatalf("agent client: %v", err)
	}

	c, err := client.Dial(client.Options{
		HostPort:  cfg.TemporalAddress,
		Namespace: cfg.TemporalNamespace,
	})
	if err != nil {
		log.Fatalf("temporal client: %v", err)
	}
	defer c.Close()

	w := worker.New(c, cfg.TemporalTaskQueue, worker.Options{})

	w.RegisterWorkflow(sewf.AgentLoopWorkflow)

	activities := sewf.NewActivities(agentClient)
	w.RegisterActivityWithOptions(activities.CallAgentActivity, activity.RegisterOptions{Name: sewf.CallAgentActivityName})
	w.RegisterActivityWithOptions(activities.RunTaskActionActivity, activity.RegisterOptions{Name: sewf.RunTaskActionActivityName})
	w.RegisterActivityWithOptions(activities.SendEmailActivity, activity.RegisterOptions{Name: sewf.SendEmailActivityName})

	slog.Info("worker starting",
		"taskQueue", cfg.TemporalTaskQueue,
		"temporalAddress", cfg.TemporalAddress,
		"llmProvider", cfg.LLMProvider,
	)
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker run: %v", err)
	}
}
