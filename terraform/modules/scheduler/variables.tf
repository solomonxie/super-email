variable "project_name" {
  type = string
}

variable "schedules" {
  description = "One EventBridge Scheduler schedule per digest, keyed by short name."
  type = map(object({
    lambda_arn = string
    expression = string # e.g. "cron(0 12 * * ? *)"
    input      = string # JSON string passed to the Lambda as its event
  }))
}
