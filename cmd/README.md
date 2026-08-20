# cmd/

One subdirectory per deployable Lambda binary — the standard Go
convention for a repo that builds several independent `main` packages
from one module (`github.com/solomonxie/super-email`, see `../go.mod`).
Each `cmd/<name>/main.go` is `package main` with its own `func main()`;
`go build ./cmd/<name>` produces exactly one binary, containing only
what that binary imports. There's no shared code yet — DESIGN.md
plans it under `/internal` (parsing, S3 stores, provider clients),
imported by full path once it exists, not a `cmd/`-local thing.

| dir               | trigger                          | does (see DESIGN.md) |
|--------------------|-----------------------------------|-----------------------|
| `email-router/`     | SES receipt rule (inbound mail)   | parse MIME, verify sender, route note commands |
| `digest-bible/`     | EventBridge Scheduler cron        | daily Bible reading digest |
| `digest-youtube/`   | EventBridge Scheduler cron        | daily YouTube uploads digest |
| `digest-social/`    | EventBridge Scheduler cron        | daily Facebook/Instagram digest |
| `digest-substack/`  | EventBridge Scheduler cron        | Substack old-post refresher digest |

Every `main.go` here is currently a placeholder: a `handler` that logs
the event and returns `nil`. Each directory name is also the Terraform
`cmd_name` that `terraform/envs/prod/lambdas.tf`'s `lambda_configs` map
passes to `modules/lambda-go` — renaming a directory means updating
that map too.

## Lambda call chain: zip → your handler function

Two triggers land here (SES, EventBridge Scheduler), but from the
`bootstrap` binary's point of view the path from "AWS decided to run
me" to "my Go code sees the event" is identical for all five. Terraform
build/deploy mechanics (how `bootstrap`/`function.zip` get produced and
redeployed) are covered in `../terraform/README.md`; this is what
happens *after* that zip is sitting in AWS, per invocation:

```
Trigger fires (SES receipt rule  |  EventBridge Scheduler cron)
        │
        ▼
AWS Lambda service — invoke function "super-email-<name>"
        │
        │  cold start only (first invocation / after a scale-out):
        │    1. fetch function.zip from Lambda's internal storage
        │    2. unzip it into /var/task
        │    3. exec /var/task/bootstrap
        │       — AWS always execs the file literally named `bootstrap`
        │         for a `provided.al2023` function; a platform
        │         convention, not something this code or Terraform
        │         decides (see ../terraform/README.md "Go build model")
        ▼
bootstrap process starts — Go's OS-level entry point
        │
func main()                              (cmd/<name>/main.go)
        │
        ▼
lambda.Start(handler)      ← aws-lambda-go takes over from here;
        │                     this call is what makes `handler` *the*
        │                     Lambda entry point (see terraform README)
        ▼
Runtime API client loop (inside lambda.Start, aws-lambda-go internals)
  GET http://${AWS_LAMBDA_RUNTIME_API}/2018-06-01/runtime/invocation/next
        │  blocks until AWS delivers this invocation's payload
        ▼
  event JSON  (SES notification, or the Scheduler's cron payload)
        │  aws-lambda-go unmarshals it into handler's 2nd param type
        │  (json.RawMessage here — raw passthrough, no typed struct yet)
        ▼
handler(ctx, event) error                ← YOUR business logic
        │                                    (cmd/<name>/main.go)
        │   placeholder today: logs the event, returns nil.
        │   this is where MIME parsing / digest fetch-render-send
        │   (see DESIGN.md) will live once it's built.
        ▼
  return value (error, or error + response for typed handlers)
        │  aws-lambda-go POSTs it back to the Runtime API
        ▼
  POST .../runtime/invocation/{id}/response   (or .../error)
        ▼
AWS Lambda service marks the invocation complete
        │
        ▼
execution environment frozen, kept warm for reuse ──▶ next invocation
  (warm start skips cold-start steps 1-3 and `main()`'s init — the
   same process loops straight back to GET .../invocation/next)
```

Three separate "entry point" layers are stacked here — the exec'd
`bootstrap` file (AWS platform convention), Go's own `func main` (OS
process entry), and `lambda.Start(handler)` (the actual Lambda
dispatch entry, owned by `aws-lambda-go`) — see
`../terraform/README.md`'s "Go build model" section for why those
three are independent and don't get to skip past each other.
