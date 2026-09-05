# super-email — Implementation plan

Design decided in [../../DESIGN.md](../../DESIGN.md); this is the build
order only. Tests live alongside each file (`_test.go`), not a
separate phase.

Priority: get the agent loop itself working and inspectable — stub
task execution, no mail edge, no notes, no digests — before wiring any
real task. That's Phase 1.

## Phase 0: Remove the old serverless scaffolding
- [ ] T0.1 Delete `terraform/` (AWS/Lambda infra, superseded by
      `deploy/docker-compose.yml`) and the 5 placeholder
      `cmd/{email-router,digest-*}/main.go` — files: `terraform/**`,
      `cmd/email-router/`, `cmd/digest-*/` — depends: none
- [ ] T0.2 Rewrite `cmd/README.md` and `terraform/README.md` (delete
      the latter) for the new layout once T0.1 and T1.* land — files:
      `cmd/README.md`, `terraform/README.md` — depends: T0.1, T1.2

## Phase 1: Bare agent loop (stub execution, no mail/tasks yet)
- [ ] T1.1 `deploy/docker-compose.yml`: Temporal (SQLite persistence)
      + Temporal UI only (no MinIO yet — nothing needs blobs at this
      phase); Ollama itself can run natively on the host (GPU access)
      instead of in Compose — just needs to be reachable at
      OLLAMA_HOST — files: `deploy/docker-compose.yml` — depends: none
- [ ] T1.2 `internal/model`: `Task`, `TaskEvent` (role: user\|agent\|system,
      content), `AgentDecision` (type: ask_user\|execute\|finish\|rework,
      question/command/params/message) — files: `internal/model/task.go`,
      `internal/model/decision.go` — depends: none
- [ ] T1.3 `internal/config`: env var loading (LLM_PROVIDER=ollama\|openai
      — default `ollama` for now, OLLAMA_HOST/OLLAMA_MODEL, OPENAI_API_KEY,
      Temporal address, max iterations, ask_user timeout) — files:
      `internal/config/config.go` — depends: none
- [ ] T1.4 `internal/agent`: `Client` interface —
      `Decide(ctx, []TaskEvent) (AgentDecision, error)` — with
      `ollama.go` (local HTTP API, build/test against this first) and
      `openai.go` (API-key backend), selected via config; shared
      prompt explains the loop contract and that `execute`/`rework`
      must name a command + params — files: `internal/agent/client.go`,
      `internal/agent/ollama.go`, `internal/agent/openai.go` —
      depends: T1.2, T1.3
- [ ] T1.5 `internal/workflow`: `AgentLoopWorkflow` + activities
      (`CallAgentActivity` wraps T1.4; `RunTaskActionActivity` is a
      dispatch table with one **stub** entry that logs the exact
      command + params and returns a canned result;
      `SendEmailActivity` is a **stub** that logs to stdout) — files:
      `internal/workflow/agent_loop.go`, `internal/workflow/activities.go`
      — depends: T1.2, T1.4
- [ ] T1.6 `cmd/worker`: registers `AgentLoopWorkflow` + activities on
      the task queue — files: `cmd/worker/main.go` — depends: T1.5
- [ ] T1.7 Manual test pass using Temporal's own CLI (no bespoke
      harness needed): `temporal workflow start` with a task
      description, `temporal workflow signal` to simulate a user
      reply, `temporal workflow show`/Temporal UI to read the decision
      trail and confirm ask/execute/finish/rework all fire correctly,
      the iteration cap holds, and the `ask_user` timeout closes the
      task — files: none (verification task) — depends: T1.1, T1.6

**Gate: don't start Phase 2 until T1.7 passes** — the loop must be
provably correct against stub output before real mail or tasks sit on
top of it.

## Phase 2: Mail edge
- [ ] T2.1 `internal/mail`: inbound webhook payload parsing (provider
      payload → `TaskEvent`) + outbound send client (provider send
      API) — files: `internal/mail/inbound.go`, `internal/mail/send.go`
      — depends: T1.2
- [ ] T2.2 `cmd/inbound-webhook`: verifies provider signature +
      `From` allow-list, resolves thread id, starts/signals
      `AgentLoopWorkflow` via Temporal client — files:
      `cmd/inbound-webhook/main.go` — depends: T2.1, T1.6
- [ ] T2.3 Swap `SendEmailActivity`'s stub for `internal/mail`'s send
      client — files: `internal/workflow/activities.go` — depends: T2.1

## Phase 3: First real task — notes
- [ ] T3.1 `internal/store`: SQLite `TaskStore` (tasks + task_events)
      and `NoteStore` (notes CRUD), embedded `schema.sql` applied on
      startup — files: `internal/store/task_store.go`,
      `internal/store/note_store.go`, `internal/store/schema.sql` —
      depends: T1.2
- [ ] T3.2 Wire `create_note`/`edit_note`/`delete_note` into
      `RunTaskActionActivity`'s dispatch table, replacing the stub for
      those commands; workflow persists `TaskEvent`s via `TaskStore` —
      files: `internal/workflow/activities.go` — depends: T3.1, T1.5

## Phase 4: Digests (parallel batch once schedules + skeleton exist)
- [ ] T4.1 `internal/store`: `DigestStateStore` (SQLite, source →
      state json) — files: `internal/store/digest_state_store.go` —
      depends: T1.2
- [ ] T4.2 `cmd/bootstrap`: one-shot, creates/updates the 4 Temporal
      Schedules, each starting `AgentLoopWorkflow` with a fixed initial
      task description — files: `cmd/bootstrap/main.go` — depends: T1.6
- [ ] T4.3 `internal/providers/bible` + register `generate_bible_digest`
      in the dispatch table — files:
      `internal/providers/bible/client.go`,
      `internal/workflow/activities.go` — depends: T4.1, T1.5
- [ ] T4.4 `internal/providers/youtube` + register
      `generate_youtube_digest` — files:
      `internal/providers/youtube/client.go`,
      `internal/workflow/activities.go` — depends: T4.1, T1.5
- [ ] T4.5 `internal/providers/substack` + register
      `generate_substack_digest` — files:
      `internal/providers/substack/client.go`,
      `internal/workflow/activities.go` — depends: T4.1, T1.5
- [ ] T4.6 `internal/providers/meta` + register
      `generate_social_digest` — files:
      `internal/providers/meta/client.go`,
      `internal/workflow/activities.go` — depends: T4.1, T1.5

T4.3-T4.6 each add a provider client (disjoint files) but all edit the
same dispatch table in `activities.go` — land them one at a time (or
split the dispatch table into one file per command to make them truly
parallel) rather than handing out 4 agents against the same file.

## Parallel execution notes
- Phase 1: T1.1-T1.3 fully parallel (3 agents); T1.4 needs T1.2+T1.3;
  T1.5 needs T1.4; T1.6 needs T1.5; T1.7 is manual verification, gates
  Phase 2.
- Phase 2 and Phase 3 can run concurrently once Phase 1's gate passes
  — disjoint files.
- Phase 4: T4.1/T4.2 parallel; T4.3-T4.6 are logically parallel but
  collide on `activities.go` — see note above.
