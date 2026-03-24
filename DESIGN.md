# super-email — Design

Personal serverless backend, in Go, that turns email into a command
interface for digests, a blog, and a notes/links inbox. Built to practice
Go: Lambda handlers, S3, EventBridge, and Terraform, running on
self-built container images rather than the zip runtime.

## 1. Goals

- Single user (me). No multi-tenant auth complexity.
- Trigger everything by emailing myself (or a dedicated inbox) from any
  provider — Gmail, Outlook, iCloud, whatever.
- Four scheduled digests: Bible reading, YouTube, Facebook/Instagram,
  Substack refresher.
- A blog: create/edit/delete/display posts.
- A notes/links inbox: capture, edit, delete.
- Everything storable in S3. Infra fully in Terraform. Lambdas run from
  container images I build and push to ECR.
- Optimize for "learn Go idioms" over "minimum services used" — but
  don't add a service without a job.

## 2. High-level architecture

```mermaid
flowchart LR
    subgraph Inbound
        ME[Me, any email provider] -->|send mail to| SESrx[SES receiving\nmy-domain.com]
        SESrx -->|store raw MIME| S3raw[(S3: raw-inbox/)]
        SESrx -->|invoke| Router[Lambda: email-router]
    end

    Router -->|parse command| Store[(S3: blogs/, notes/)]
    Router -->|reply| SEStx[SES sending]
    SEStx --> ME

    subgraph Scheduled digests
        EB[EventBridge Scheduler\n4 cron rules] --> DBible[Lambda: digest-bible]
        EB --> DYT[Lambda: digest-youtube]
        EB --> DSocial[Lambda: digest-social]
        EB --> DSub[Lambda: digest-substack]
    end
    DBible & DYT & DSocial & DSub -->|compose + send| SEStx
    DYT -.->|API| YT[YouTube Data API]
    DSocial -.->|API| Meta[Meta Graph API]
    DSub -.->|RSS/HTML| Substack[Substack feeds]
    DBible -.->|API| BibleAPI[Bible API]

    subgraph Blog site
        APIGW[API Gateway HTTP API] --> BlogAPI[Lambda: blog-api]
        BlogAPI --> Store
        Reader[Browser] --> APIGW
    end

    SSM[(SSM Parameter Store\nsecrets/config)] -.-> Router
    SSM -.-> DYT
    SSM -.-> DSocial
    SSM -.-> DSub
```

Every Lambda is a separate container image (its own `cmd/` binary), same
ECR repo family, versioned by image tag. No shared "fat" function.

## 3. Inbound email

**Decision: AWS SES email receiving on a domain I own, not IMAP
polling of a Gmail/Outlook inbox.**

Rationale: SES receiving is the AWS-native path — it can invoke a
Lambda directly (or drop to S3 and fire an S3 event), needs no stored
IMAP/OAuth credentials, and is the whole reason to reach for SES in a
"practice AWS" project. The tradeoff is I need a domain with MX records
pointed at SES (any cheap domain works; a subdomain like
`inbox.mydomain.com` is enough — I don't need to move my main mail
there). *I* still send from Gmail/Outlook/whatever; only the receiving
address lives on SES.

Flow:
1. SES receipt rule on `me@inbox.mydomain.com` (and maybe
   `blog@…`, `note@…` as friendlier aliases — same rule, routing by
   recipient).
2. Rule action: store raw MIME in `s3://<bucket>/raw-inbox/<message-id>`,
   then invoke `email-router` Lambda with the S3 location (SES's
   Lambda action gives the mail object in the event, avoiding a second
   S3-trigger hop).
3. `email-router`:
   - Verifies `From` is my allow-listed address (hard requirement —
     SES receiving is otherwise open to anyone who knows the address;
     spam/abuse must be rejected before any processing). Anything else
     is dropped and logged, no reply sent.
   - Parses MIME (Go: `net/mail` + `mime/multipart`) — subject, plain
     text/HTML body, links found in the body.
   - Routes by command grammar (§5) to blog or notes handler.
   - Sends a reply via SES (confirmation + link, or error) to the
     original sender.

Outbound (digests + replies) uses the same SES identity, sending
domain verified + DKIM configured via Terraform (`aws_ses_domain_identity`,
`aws_ses_domain_dkim`).

## 4. Storage model

**Decision: S3 only for v1, no DynamoDB.** Single writer, personal
scale, and it keeps the Terraform/IAM surface small. Layout:

```
s3://super-email-data/
  raw-inbox/<message-id>.eml          # raw MIME, TTL'd via lifecycle rule (30d)
  blogs/
    posts/<slug>.json                 # {id, slug, title, body_md, tags, status, created_at, updated_at}
    index.json                        # [{slug, title, status, updated_at}, ...] sorted by updated_at desc
  notes/
    items/<id>.json                   # {id, type: note|link, content, url?, tags, created_at}
    index.json                        # [{id, type, summary, created_at}, ...]
  digest-state/
    bible-progress.json               # {day: n} — resume point for reading plan
    substack-seen.json                # {slug: [seen post ids]} — avoid repeats
```

Each write (create/edit/delete) rewrites the affected item object *and*
the relevant `index.json` in the same handler call — not a queue, not
eventual consistency. S3 offers no multi-object transactions, so
concurrent writers could race, but there's only one writer (me, via
email) so this is fine. `index.json` is small enough to read-modify-write
whole; if it gets unwieldy, revisit (see §9).

IDs: content-addressed-ish — `slug` for blogs (slugified title,
de-duplicated with a numeric suffix on collision), ULID for notes (sortable,
no clock sync issues).

## 5. Email command grammar

Subject-line driven, case-insensitive prefix match, body is the payload:

| Subject prefix              | Action                                   | Body                     |
|------------------------------|-------------------------------------------|--------------------------|
| `blog new: <title>`          | create post, status=draft                | markdown body            |
| `blog publish: <slug>`       | flip status draft→published              | (ignored)                |
| `blog edit: <slug>`          | replace body                             | new markdown body        |
| `blog delete: <slug>`        | delete post + index entry                | (ignored)                |
| `note: <anything>`           | new note; if body/subject is a bare URL, `type=link` | text or nothing |
| `note edit: <id>`            | replace content                          | new text                 |
| `note delete: <id>`          | delete note                              | (ignored)                |
| *(no recognized prefix)*     | default: treat whole mail as a new note  | subject+body captured    |

Reply always confirms: `"Created blog post 'my-title' → https://blog.mydomain.com/my-title"`
or an error explaining the parse failure. This makes the router
forgiving — an unrecognized command becomes a note instead of silently
failing, since "capture everything" is the fallback goal of a
send-yourself-stuff inbox.

## 6. Digest pipelines

One Lambda per digest (separate container images, separate EventBridge
Scheduler cron rules), sharing an `internal/digest` package for the
"fetch → render text/HTML → send via SES" skeleton. Per-source notes:

- **Bible read** — `internal/providers/bible`, e.g. bible-api.com (free,
  no key). State in `digest-state/bible-progress.json` tracks a
  sequential reading-plan day counter; digest = today's reading + link.
- **YouTube digest** — YouTube Data API v3, `playlistItems.list` on each
  subscribed channel's uploads playlist (channel list is config, not
  auto-subscribed — avoids needing OAuth on my personal account; an API
  key + a hardcoded channel-ID list is enough). Quota cost noted in
  README once implemented (10k units/day free tier; `playlistItems.list`
  is cheap, `search.list` is not — avoid the latter).
- **Facebook/Instagram digest** — Meta Graph API. Flagged as the
  highest-friction integration: needs a Meta developer app, a
  long-lived (60-day) user/page token that must be refreshed
  out-of-band (no fully-automatable refresh without a login flow), and
  Instagram's API increasingly requires a Business/Creator account
  linked to a Facebook Page. Treat as best-effort; if token refresh
  becomes too annoying, fall back to whatever public RSS bridge is
  available at implementation time, or drop Instagram and keep just
  Facebook Page posts (Pages have simpler long-lived tokens).
- **Substack refresher** — "old posts" means the RSS feed
  (`<pub>.substack.com/feed`) alone isn't enough — it only carries
  recent items. Use the feed to seed known posts, and Substack's
  sitemap (`<pub>.substack.com/sitemap.xml`) or archive page to
  discover the full back-catalog once, then randomly resurface one
  not-recently-sent post per run, tracked in
  `digest-state/substack-seen.json`.

## 7. Blog subsystem

`blog-api` Lambda behind API Gateway (HTTP API, cheaper than REST API).
Two responsibilities in one function for now (split later if it grows):

- **Write path**: not exposed over HTTP at all in v1 — creation/edit/
  delete only happens via email (§5). Keeps the public API surface
  read-only and avoids needing auth on API Gateway for v1.
- **Read path**: `GET /` (index, published only), `GET /{slug}`
  (single post, markdown rendered to HTML server-side —
  `github.com/yuin/goldmark` is a reasonable Go choice). Drafts are
  never served over HTTP, only visible via the email reply link's
  slug if I fetch them a different way (or just don't — drafts stay
  S3-only until published).

No CloudFront/custom domain in v1 — API Gateway's default endpoint is
enough to read from a browser; add a custom domain + CloudFront later
if it needs to feel like a "real" public blog.

## 8. Notes/links subsystem

No HTTP surface in v1 — email is the only interface (capture via plain
send, edit/delete via the `note edit:`/`note delete:` commands, §5).
Listing/searching notes, if wanted later, is a `notes-api` Lambda
mirroring blog-api's read path — deferred until it's actually painful
to grep S3 by hand.

## 9. Go project layout

```
/cmd
  email-router/       main.go — SES-invoked entrypoint
  digest-bible/
  digest-youtube/
  digest-social/       (Facebook + Instagram together)
  digest-substack/
  blog-api/            API Gateway entrypoint
/internal
  email/               MIME parsing, SES send helpers, command grammar parser
  store/                S3-backed repositories: BlogStore, NoteStore, DigestStateStore
  digest/               shared "fetch → render → send" skeleton + email templates
  providers/
    bible/
    youtube/
    meta/
    substack/
  model/                Blog, Note, Digest types
  config/               env var + SSM parameter loading
/terraform
  modules/
    lambda-container/   generic module: ECR repo + Lambda from image URI + IAM role
    ses/                 domain identity, DKIM, receipt rule set, MAIL FROM
    s3/
    eventbridge/         scheduler rules → Lambda targets
    apigateway/
  envs/
    prod/
Dockerfile               single multi-stage Dockerfile, ARG CMD selects which /cmd/* to build
```

One shared `Dockerfile` with a build arg (`docker build --build-arg
CMD=email-router`) rather than five near-duplicate Dockerfiles — each
Lambda still gets its own ECR repo/image tag from Terraform, just built
from the same recipe.

## 10. Infra & security notes

- **Secrets/config**: SSM Parameter Store (`SecureString` for tokens),
  not Secrets Manager — free, and rotation isn't automatable here
  anyway (Meta tokens especially need manual refresh). Params:
  allow-listed sender address, YouTube API key + channel IDs, Meta
  token, Substack publication list.
- **IAM**: one role per Lambda, least privilege — e.g. `email-router`
  gets `s3:GetObject` on `raw-inbox/*` + `s3:PutObject/GetObject` on
  `blogs/*` and `notes/*`, `ses:SendEmail`; `digest-youtube` gets no S3
  blog/note access at all, just `digest-state/*` + `ses:SendEmail`.
- **Abuse guard**: SES receipt rule scoped to specific recipient
  addresses only; router double-checks `From` against the allow-list
  before doing anything. Everything else is dropped, not bounced (don't
  confirm the address exists to a spammer).
- **Observability**: structured logs via `log/slog` (JSON handler) —
  Lambda ships these to CloudWatch automatically. A CloudWatch alarm on
  any Lambda's `Errors` metric → SNS → email, so failures surface
  without polling logs.
- **Container images**: multi-stage build, `FROM public.ecr.aws/lambda/provider-al2023`
  as the runtime base, static Go binary copied in — avoids CGO/musl
  headaches and is the standard pattern for Go-on-Lambda-containers.

## 11. Open questions / deferred

- DynamoDB migration path if S3 index-file writes ever race or the
  index grows past comfortable read-modify-write size (not expected at
  personal scale, but the repository interface in `internal/store`
  should stay swappable).
- Meta token refresh automation — likely stays manual (calendar
  reminder) unless a clean unattended refresh flow exists.
- Whether blog gets a custom domain/CloudFront, or stays on the raw API
  Gateway URL.
- Whether notes get a read API, or S3 console browsing is good enough.

## 12. Suggested build order

1. Terraform bootstrap: S3 bucket, SES domain identity + DKIM (this has
   DNS propagation lag — start it first).
2. `email-router` + SES receipt rule + `store` package (blog/notes
   S3 repos) + command grammar. Prove the full inbound loop with a
   dumb "note" fallback before building blog commands.
3. `blog-api` read path, since it reuses `store` and gives something
   visible quickly.
4. One digest end-to-end (Bible — simplest API, no auth) to prove the
   EventBridge → Lambda → SES send path, then repeat for YouTube,
   Substack, Meta in roughly that order of API friction.
