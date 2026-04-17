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

Send an email, get something done. Subject line is the command, body
is the content:

```
To:      note@inbox.example.com
Subject: A link worth keeping
Body:    https://example.com/some-article
```

No recognized command? The email is just saved as a note — nothing is
ever silently dropped.

Full command reference lives in [DESIGN.md § Email command grammar](./DESIGN.md#5-email-command-grammar).

## Under the hood

Serverless, on AWS, written in Go:

- **AWS Lambda** (Go binaries, zip-deployed — no Docker, no container
  registry) for every handler — email routing and each digest.
- **Amazon SES** for receiving and sending mail.
- **Amazon S3** as the only datastore — notes and digest state, no
  database.
- **Amazon EventBridge Scheduler** to trigger the daily digests.
- **Terraform** for all infrastructure, no console clicking.
- No API Gateway, no public endpoint of any kind — email in, email out.

See [DESIGN.md](./DESIGN.md) for the architecture diagram, data model,
and per-feature notes.

## Status

Early design phase — nothing is deployed yet. Follow along in
[DESIGN.md](./DESIGN.md#11-suggested-build-order) for the current build
order.
