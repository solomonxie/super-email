# Depends on: nothing (leaf module) — callers pass in bucket ARNs etc.
#   via iam_statements/environment.
# Depended on by: envs/prod's `lambdas` module (one instance per
#   cmd/<name>), whose outputs feed ses.tf and scheduler.tf.
#
# null_resource.build (go build, local-exec) ──→ data.archive_file.this (zip it)
#                                                        │
# aws_iam_role.this ──→ aws_iam_role_policy_attachment.logs                │
#                    └─→ aws_iam_role_policy.extra (var.iam_statements)    │
# aws_cloudwatch_log_group.this                                           │
#                                                        ▼
#                                       aws_lambda_function.this (zip deploy,
#                                         runtime = provided.al2023)
#
# One IAM role + one Lambda per module instance, so IAM stays
# least-privilege per function (per DESIGN.md) instead of one shared fat
# role. No Docker/ECR: each function is a single static `bootstrap`
# binary, cross-compiled and zipped locally.

locals {
  full_name = "${var.project_name}-${var.name}"
  build_dir = "${var.source_dir}/.build/${var.name}"
  go_arch   = var.architecture == "arm64" ? "arm64" : "amd64"
}

# Rebuilds whenever go.mod or any .go file under source_dir changes.
# Requires Go on the machine running `terraform apply`.
resource "null_resource" "build" {
  triggers = {
    gomod_sha  = filesha256("${var.source_dir}/go.mod")
    source_sha = sha1(join("", [for f in sort(fileset(var.source_dir, "**/*.go")) : filesha256("${var.source_dir}/${f}")]))
    cmd_name   = var.cmd_name
    go_arch    = local.go_arch
  }

  provisioner "local-exec" {
    working_dir = var.source_dir
    command     = <<-EOT
      set -euo pipefail
      mkdir -p ${local.build_dir}
      CGO_ENABLED=0 GOOS=linux GOARCH=${local.go_arch} \
        go build -tags lambda.norpc -o ${local.build_dir}/bootstrap ./cmd/${var.cmd_name}
    EOT
  }
}

# Deferred to apply-time via depends_on, so the first `terraform plan`
# (before the binary exists) doesn't fail trying to hash a missing file.
data "archive_file" "this" {
  type        = "zip"
  source_file = "${local.build_dir}/bootstrap"
  output_path = "${local.build_dir}/function.zip"
  depends_on  = [null_resource.build]
}

resource "aws_iam_role" "this" {
  name = "${local.full_name}-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "logs" {
  role       = aws_iam_role.this.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy" "extra" {
  count = length(var.iam_statements) > 0 ? 1 : 0
  name  = "${local.full_name}-extra"
  role  = aws_iam_role.this.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [for s in var.iam_statements : {
      Sid      = s.sid
      Effect   = "Allow"
      Action   = s.actions
      Resource = s.resources
    }]
  })
}

resource "aws_cloudwatch_log_group" "this" {
  name              = "/aws/lambda/${local.full_name}"
  retention_in_days = var.log_retention_days
}

resource "aws_lambda_function" "this" {
  function_name = local.full_name
  description   = var.description
  role          = aws_iam_role.this.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = [var.architecture]

  filename         = data.archive_file.this.output_path
  source_code_hash = data.archive_file.this.output_base64sha256

  timeout     = var.timeout
  memory_size = var.memory_size

  environment {
    variables = var.environment
  }

  depends_on = [
    aws_iam_role_policy_attachment.logs,
    aws_cloudwatch_log_group.this,
  ]
}
