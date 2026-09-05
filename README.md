# super-email

> 🚧 **Work in progress.** This is a design/build-in-progress personal
> project — not deployed, not usable yet. See [DESIGN.md](./DESIGN.md)
> for the full technical plan and build order.

Your inbox as a command line. Email yourself (from any provider —
Gmail, Outlook, iCloud, whatever) to run digests and capture notes and
links — no app to install, no website to log into.

## What it does

- **Daily digests, delivered to your inbox**
  - Bible reading plan
  - YouTube — new uploads from your channels
  - Facebook / Instagram digest
  - Substack refresher — resurfaces old posts from your subscriptions
- **Notes & links** — send yourself a thought or a URL, edit or delete
  it later, all by email.

## How it works

Send an email, get something done — no fixed subject-line syntax. An
LLM agent reads the email and decides what you mean: ask a follow-up
question if it's unclear, do the task, or redo it if the result isn't
good enough.

```
To:      me@inbox.example.com
Subject: A link worth keeping
Body:    https://example.com/some-article
```

Can't tell what you meant? It's saved as a note — nothing is ever
silently dropped.

See [DESIGN.md: the agent loop](./DESIGN.md#4-the-agent-loop) for how
that decision loop works.

## Under the hood

Self-hosted, written in Go, orchestrated by an LLM agent loop:

- Each request (an inbound email, or a scheduled digest tick) becomes
  one **Temporal** workflow run — the agent loop itself: ask a
  clarifying question, run the task, decide it's done, or redo it.
- A hosted email API handles send/receive (deliverability, DKIM, spam
  filtering) — everything else runs on infra I own.
- **SQLite** for tasks, notes, and digest state; **MinIO** for raw
  mail/attachments.
- No API Gateway, no AWS — a single Docker Compose stack.

See [DESIGN.md](./DESIGN.md) for the architecture diagram, data model,
and per-feature notes.

## Status

Early design phase — nothing is deployed yet. Follow along in
[DESIGN.md](./DESIGN.md#11-suggested-build-order) for the current build
order.
