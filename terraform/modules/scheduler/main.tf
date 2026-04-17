# Depends on: lambda-container instances for each digest (lambda_arn per
#   entry in var.schedules), passed in by envs/prod.
# Depended on by: nothing (leaf of the graph on the digest side).
#
# aws_iam_role.scheduler ──→ aws_iam_role_policy.invoke (lambda:InvokeFunction
#   on every schedule's lambda_arn) ──→ aws_scheduler_schedule.this (for_each)
#
# EventBridge Scheduler invokes via an assumed IAM role, not a
# service-principal resource policy — no aws_lambda_permission needed
# here (unlike SES/API Gateway).

resource "aws_iam_role" "scheduler" {
  name = "${var.project_name}-scheduler-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "scheduler.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy" "invoke" {
  name = "${var.project_name}-scheduler-invoke"
  role = aws_iam_role.scheduler.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "lambda:InvokeFunction"
      Resource = [for s in var.schedules : s.lambda_arn]
    }]
  })
}

resource "aws_scheduler_schedule" "this" {
  for_each            = var.schedules
  name                = "${var.project_name}-digest-${each.key}"
  schedule_expression = each.value.expression

  flexible_time_window {
    mode = "OFF"
  }

  target {
    arn      = each.value.lambda_arn
    role_arn = aws_iam_role.scheduler.arn
    input    = each.value.input
  }
}
