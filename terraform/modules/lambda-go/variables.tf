variable "project_name" {
  type        = string
  description = "Prefix for all resource names, e.g. \"super-email\"."
}

variable "name" {
  type        = string
  description = "Short name for this Lambda, e.g. \"email-router\". Combined with project_name for resource names."
}

variable "cmd_name" {
  type        = string
  description = "Directory under /cmd to build, e.g. \"email-router\" builds ./cmd/email-router."
}

variable "description" {
  type    = string
  default = ""
}

variable "source_dir" {
  type        = string
  description = "Repo root — contains go.mod and cmd/. Build runs with this as the working directory."
}

variable "architecture" {
  type        = string
  default     = "arm64"
  description = "Lambda architecture; the Go build's GOARCH is derived from this."
}

variable "timeout" {
  type    = number
  default = 30
}

variable "memory_size" {
  type    = number
  default = 256
}

variable "environment" {
  type    = map(string)
  default = {}
}

variable "log_retention_days" {
  type    = number
  default = 14
}

variable "iam_statements" {
  description = "Extra IAM policy statements for this function's role, beyond basic CloudWatch Logs access."
  type = list(object({
    sid       = string
    actions   = list(string)
    resources = list(string)
  }))
  default = []
}
