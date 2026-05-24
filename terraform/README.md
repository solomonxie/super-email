# Terraform

Provisions the AWS side of super-email (design: [../DESIGN.md](../DESIGN.md)):
one S3 data bucket, five zip-deployed Lambdas (Terraform itself runs
`go build` and zips the result — no Docker, no container registry), SES
inbound receiving, and four EventBridge Scheduler digests. No API
Gateway or any other public HTTP surface — every interaction is by
email. File order in `envs/prod/` doesn't matter — reference order
does; each file's head comment states what it depends on. Modules live
in `modules/`, one per AWS-facing concern, wired together in `envs/prod/`.

```
terraform apply   (from terraform/envs/prod)
  ├─ providers.tf          → AWS + archive providers, local state (for now)
  ├─ variables.tf          → resolve inputs (terraform.tfvars)
  ├─ data.tf                → account id, for IAM/permission ARNs
  ├─ s3.tf                  → module.data_bucket
  │                             (modules/s3-data: bucket, encryption,
  │                              versioning, raw-inbox/ 30d lifecycle)
  ├─ lambdas.tf             → module.lambdas["email-router" | "digest-bible" |
  │                             "digest-youtube" | "digest-social" |
  │                             "digest-substack"]  (for_each)
  │                             (modules/lambda-go, ×5: go build [cmd/<name>]
  │                              → zip → IAM role → Lambda; off module.data_bucket)
  ├─ ses.tf                 → module.ses_inbound
  │                             (modules/ses-inbound: domain identity, DKIM,
  │                              receipt rule → S3 + email-router Lambda;
  │                              off module.data_bucket, module.lambdas)
  ├─ scheduler.tf           → module.digest_schedules
  │                             (modules/scheduler: 4 cron schedules →
  │                              digest-* Lambdas; off module.lambdas)
  └─ outputs.tf             → bucket name, DNS records to set manually if
        │                      route53_zone_id is unset
        ▼
point ses_domain's MX record at SES   (manual, once — at your registrar
                                        or automatic if route53_zone_id
                                        was set; see outputs)
        ▼
send yourself mail / write Go code, then `terraform apply` again
  to rebuild and redeploy any Lambda whose source changed
```

## Prerequisites

- Terraform >= 1.7, AWS credentials in the environment.
- Go installed — `lambdas.tf`'s build step shells out to `go build`
  directly (see `modules/lambda-go/main.tf`); no Docker, no ECR, no
  registry login required. Also run `go mod tidy` once locally to
  commit a `go.sum`.
- A domain (or subdomain) you control, for `ses_domain`. It does not
  need to be your everyday inbox — a subdomain like `inbox.example.com`
  is enough; you keep sending from Gmail/Outlook/whatever.

## First run

```sh
cp envs/prod/terraform.tfvars.example envs/prod/terraform.tfvars
# edit terraform.tfvars: data_bucket_name, ses_domain, allowed_sender_email

cd envs/prod
terraform init
terraform apply
```

Or, from the repo root, `make init` / `make deploy` / `make destroy` —
see the [Makefile](../Makefile) for these and other common commands
(`build`, `test`, `fmt`, `logs-<lambda-name>`).

Every `cmd/*` currently builds to a "hello world" placeholder handler
(see [../DESIGN.md](../DESIGN.md#11-suggested-build-order)) — `apply`
stands up real infrastructure end-to-end so Go development can happen
against it from the start, without a separate "wire up AWS" pass later.

## Go build model

One Go module (`go.mod`) for the whole repo; each `cmd/<name>/main.go`
is its own `package main` — the standard layout for building several
independent binaries from one module. `go build ./cmd/email-router`
builds only that binary. There's no `lib/`/`internal/` yet — once
shared code exists it's a plain importable package outside `cmd/`
(e.g. `lib/parser`, imported as
`github.com/solomonxie/super-email/lib/parser`); it needs no deploy
step of its own, since Go's linker keeps only what each binary
actually imports — five Lambdas sharing one `lib/` package still
produce five independent `bootstrap` binaries.

Three different "entry points" are involved, at three different layers:

1. **OS process entry** — `func main()` in each `cmd/<name>/main.go`,
   ordinary Go.
2. **Lambda's Go-level entry, decided by the `aws-lambda-go` library**
   — `lambda.Start(handler)` called inside `main()` is what makes
   `handler` *the* Lambda entry point: it opens the AWS Lambda Runtime
   API polling loop and dispatches each invocation's event to whatever
   function was passed in. This is the library's doing, not Terraform's
   or AWS's — nothing outside the Go binary knows or cares which
   function `handler` is; the binary just presents itself to AWS as one
   opaque process.
3. **AWS's entry into the zip is separate, and *not* decided by
   `aws-lambda-go`** — `modules/lambda-go/main.tf` builds
   `GOOS=linux GOARCH=<arch> go build -o bootstrap ./cmd/<name>` and
   zips just that file. `runtime = "provided.al2023"` is AWS's
   "bring-your-own-runtime" option (no interpreter baked in), and for
   any `provided`/`provided.al2023` runtime AWS *always* execs the
   file literally named `bootstrap` at the zip's root, before any Go
   code — including `aws-lambda-go` — has run. That's a platform
   convention enforced by the Lambda execution environment itself,
   which is why the build's `-o` is hardcoded to that name. The
   `handler = "bootstrap"` attribute on `aws_lambda_function.this` is
   required by the resource schema, but its value is otherwise unused
   here — `lambda.Start()` always serves whichever Go function it was
   given, regardless of the `_HANDLER` env var AWS sets from it.

So the real call chain is: AWS execs `bootstrap` → Go's `main()` runs
→ `lambda.Start(handler)` opens the runtime loop and calls `handler`
per invocation.

## Notes

- No CloudWatch alarms/SNS yet (DESIGN.md §9 mentions them as a
  follow-up) — errors are visible in each Lambda's CloudWatch Logs
  group for now.
- Each Lambda's `bootstrap` binary and `function.zip` land in
  `.build/<name>/` at the repo root (gitignored, rebuilt on demand from
  `go.mod`/`.go` file hashes — see `modules/lambda-go/main.tf`'s
  `null_resource.build` triggers).
- `route53_zone_id` is optional. If your domain's zone isn't in this
  AWS account (or isn't in Route53 at all), leave it `null` and use the
  `ses_*` outputs to create the TXT/CNAME/MX records at your registrar.
- State is local (`terraform.tfstate`, gitignored) until a remote
  backend is set up — fine solo, worth moving to S3 before this has
  more than one contributor.
