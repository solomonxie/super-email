# cmd/

One subdirectory per `main` package (repo module:
`github.com/solomonxie/super-email`, see `../go.mod`) — the standard Go
convention for a repo with several independent binaries. No Lambda here
anymore (see DESIGN.md — EC2 + Docker Compose replaced Lambda +
EventBridge); each binary is a long-running or one-shot process on the
box.

| dir                | runs as                     | does (see DESIGN.md) |
|---------------------|------------------------------|-----------------------|
| `worker/`           | long-running (Docker Compose)| Temporal worker: registers `AgentLoopWorkflow` + its activities, polls the task queue |
| `inbound-webhook/`  | long-running (Docker Compose), not yet built | HTTP server behind Caddy; verifies SNS + sender, starts/signals `AgentLoopWorkflow` (Phase 2) |
| `bootstrap/`        | one-shot, not yet built      | creates/updates the 4 Temporal Schedules for digests (Phase 4) |

`internal/` holds the actual logic (agent client, workflow, mail,
store, providers — DESIGN.md §8); `cmd/*` binaries just wire it up and
call `w.Run()`/`http.ListenAndServe()`/exit.

## worker: from `docker compose up` to a running workflow

```
docker compose up -d              (deploy/docker-compose.yml)
        │
        ▼
cmd/worker's main()
  1. config.Load()                 — env vars: LLM_PROVIDER, TEMPORAL_*, ...
  2. agent.NewClient(cfg)          — Ollama or OpenAI backend
  3. client.Dial(...)              — connect to the Temporal server
  4. worker.New(..., taskQueue)
  5. RegisterWorkflow(AgentLoopWorkflow)
  6. RegisterActivityWithOptions(...) × 3   — CallAgent/RunTaskAction/SendEmail
  7. w.Run(worker.InterruptCh())    — blocks, polling the task queue
        │
        ▼
Temporal server dispatches a workflow/activity task
        │
        ▼
AgentLoopWorkflow / Activities methods run   (internal/workflow)
```

Something has to *start* a workflow execution for the worker to have
anything to poll — for now that's the Temporal CLI (`temporal workflow
start ...`, see `../docs/design/super-email-plan.md` T1.7); once Phase 2
lands, `inbound-webhook` does that from real mail, and Phase 4's
`bootstrap` does it on a schedule.
