# Terraform

Provisions the AWS side of super-email (design: [../DESIGN.md](../DESIGN.md)):
one EC2 instance to run the app (Temporal + worker via Docker Compose,
per `deploy/docker-compose.yml`), one S3 data bucket, and SES for
inbound + outbound mail. Terraform's job stops at *infra* — it doesn't
build or deploy the Go app (unlike the old Lambda-based design, which
had Terraform run `go build`/zip/upload); getting the app onto the box
is a manual `ssh` + `docker compose up -d` step, same as it would be on
a hand-provisioned VPS. File order in `envs/prod/` doesn't matter —
reference order does; each file's head comment states what it depends
on. Modules live in `modules/`, one per AWS-facing concern, wired
together in `envs/prod/`.

```
terraform apply   (from terraform/envs/prod)
  ├─ providers.tf          → AWS provider, local state (for now)
  ├─ variables.tf          → resolve inputs (terraform.tfvars)
  ├─ data.tf                → account id, for IAM/permission ARNs
  ├─ s3.tf                  → module.data_bucket
  │                             (modules/s3-data: bucket, encryption,
  │                              versioning, raw-inbox/ 30d lifecycle,
  │                              attachments/ kept indefinitely)
  ├─ ec2.tf                 → module.app_host
  │                             (modules/ec2-host: default-VPC instance,
  │                              security group, IAM instance role
  │                              (S3 + SES), EIP, user_data installs
  │                              Docker + Compose + Caddy only)
  ├─ ses.tf                 → module.ses_inbound
  │                             (modules/ses-inbound: domain identity, DKIM,
  │                              receipt rule → S3 + SNS topic → HTTPS
  │                              subscription at module.app_host's
  │                              webhook URL; off module.data_bucket,
  │                              module.app_host)
  └─ outputs.tf             → bucket name, instance IP/SSH command,
        │                      webhook URL, DNS records to set manually
        │                      if route53_zone_id is unset
        ▼
point ses_domain's MX record at SES, and the webhook subdomain's A
record at the instance IP   (manual, once — at your registrar, or
automatic if route53_zone_id was set; see outputs)
        ▼
ssh onto the instance, clone the repo, `docker compose up -d`
  (deploy/docker-compose.yml) — this confirms the SNS HTTPS
  subscription (inbound-webhook must answer the SubscribeURL handshake)
        ▼
request SES production access (new accounts are sandboxed — can only
  send to verified addresses; console → SES → Account dashboard)
        ▼
send yourself mail
```

## Prerequisites

- Terraform >= 1.7, AWS credentials in the environment.
- An SSH keypair (e.g. `ssh-keygen -t ed25519`) — its public key goes
  into `ec2_ssh_public_key`.
- A domain (or subdomain) you control, for `ses_domain`. It does not
  need to be your everyday inbox — a subdomain like `inbox.example.com`
  is enough; you keep sending from Gmail/Outlook/whatever if you want,
  though outbound in this design goes through SES too.

## First run

```sh
cp envs/prod/terraform.tfvars.example envs/prod/terraform.tfvars
# edit terraform.tfvars: data_bucket_name, ses_domain, ec2_ssh_public_key,
# ssh_allowed_cidr

cd envs/prod
terraform init
terraform apply
```

Or, from the repo root, `make init` / `make deploy` / `make destroy` —
see the [Makefile](../Makefile).

## Manual one-time steps

Two things stay manual regardless of `route53_zone_id`:

- **SES production access.** A new SES account is sandboxed — it can
  only send to addresses you've individually verified in the console.
  Request production access once (SES console → Account dashboard) or
  every outbound send will fail until you do.
- **The SNS subscription confirmation.** `aws_sns_topic_subscription`
  creates the subscription in `PendingConfirmation` state; SNS then
  GETs the `SubscribeURL` it sends to `webhook_url`. It only flips to
  `Confirmed` once `inbound-webhook` is actually running and answers
  that request — deploy the app before expecting inbound mail to work,
  and re-check the subscription's status in the SNS console if it
  doesn't.

If `route53_zone_id` is null, also create the DNS records `outputs.tf`
prints (SES verification TXT, DKIM CNAMEs, inbound MX, and the webhook
subdomain's A record) at your registrar.

## Notes

- No CloudWatch alarms/SNS-for-ops yet — errors are visible via
  Temporal's own UI (workflow/activity failures) and the app's logs on
  the instance for now.
- State is local (`terraform.tfstate`, gitignored) until a remote
  backend is set up — fine solo, worth moving to S3 before this has
  more than one contributor.
- Digests are Temporal Schedules created by `cmd/bootstrap`
  (`DESIGN.md` §6) — nothing digest-shaped lives in Terraform; there's
  no EventBridge Scheduler here.
