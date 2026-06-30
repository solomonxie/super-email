# Depends on: s3.tf (module.data_bucket), variables.tf.
# Depended on by: ses.tf, scheduler.tf (each reads module.lambdas["<name>"]
#   outputs for the function it wires up).
#
# local.lambda_configs describes the 5 functions from DESIGN.md; the
# `lambdas` module below fans them all out through modules/lambda-go via
# for_each, one IAM role + zip-deployed Lambda each (no ECR, no Docker).

locals {
  common_env = {
    DATA_BUCKET = module.data_bucket.bucket_name
  }

  repo_root = abspath("${path.module}/../../..")

  lambda_configs = {
    email-router = {
      cmd_name    = "email-router"
      description = "Parses inbound SES mail, verifies sender, routes note commands"
      timeout     = 30
      memory_size = 256
      environment = merge(local.common_env, { ALLOWED_SENDER = var.allowed_sender_email })
      iam_statements = [
        {
          sid       = "ReadRawInbox"
          actions   = ["s3:GetObject"]
          resources = ["${module.data_bucket.bucket_arn}/raw-inbox/*"]
        },
        {
          sid       = "RWNotes"
          actions   = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
          resources = ["${module.data_bucket.bucket_arn}/notes/*"]
        },
        {
          sid       = "SendMail"
          actions   = ["ses:SendEmail", "ses:SendRawEmail"]
          resources = ["*"]
        },
      ]
    }

    digest-bible = {
      cmd_name    = "digest-bible"
      description = "Sends the daily Bible reading digest"
      timeout     = 30
      memory_size = 256
      environment = local.common_env
      iam_statements = [
        {
          sid       = "DigestState"
          actions   = ["s3:GetObject", "s3:PutObject"]
          resources = ["${module.data_bucket.bucket_arn}/digest-state/*"]
        },
        {
          sid       = "SendMail"
          actions   = ["ses:SendEmail", "ses:SendRawEmail"]
          resources = ["*"]
        },
      ]
    }

    digest-youtube = {
      cmd_name    = "digest-youtube"
      description = "Sends the daily YouTube uploads digest"
      timeout     = 60
      memory_size = 256
      environment = local.common_env
      iam_statements = [
        {
          sid       = "DigestState"
          actions   = ["s3:GetObject", "s3:PutObject"]
          resources = ["${module.data_bucket.bucket_arn}/digest-state/*"]
        },
        {
          sid       = "SendMail"
          actions   = ["ses:SendEmail", "ses:SendRawEmail"]
          resources = ["*"]
        },
      ]
    }

    digest-social = {
      cmd_name    = "digest-social"
      description = "Sends the daily Facebook/Instagram digest"
      timeout     = 60
      memory_size = 256
      environment = local.common_env
      iam_statements = [
        {
          sid       = "DigestState"
          actions   = ["s3:GetObject", "s3:PutObject"]
          resources = ["${module.data_bucket.bucket_arn}/digest-state/*"]
        },
        {
          sid       = "SendMail"
          actions   = ["ses:SendEmail", "ses:SendRawEmail"]
          resources = ["*"]
        },
      ]
    }

    digest-substack = {
      cmd_name    = "digest-substack"
      description = "Sends the Substack old-post refresher digest"
      timeout     = 60
      memory_size = 256
      environment = local.common_env
      iam_statements = [
        {
          sid       = "DigestState"
          actions   = ["s3:GetObject", "s3:PutObject"]
          resources = ["${module.data_bucket.bucket_arn}/digest-state/*"]
        },
        {
          sid       = "SendMail"
          actions   = ["ses:SendEmail", "ses:SendRawEmail"]
          resources = ["*"]
        },
      ]
    }
  }
}

module "lambdas" {
  for_each = local.lambda_configs
  source   = "../../modules/lambda-go"

  project_name   = var.project_name
  name           = each.key
  cmd_name       = each.value.cmd_name
  description    = each.value.description
  source_dir     = local.repo_root
  timeout        = each.value.timeout
  memory_size    = each.value.memory_size
  environment    = each.value.environment
  iam_statements = each.value.iam_statements

  log_retention_days = var.log_retention_days
}
