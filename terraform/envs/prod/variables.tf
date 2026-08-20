# Depends on: nothing.
# Depended on by: every other file in this env.

variable "project_name" {
  type    = string
  default = "super-email"
}

variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "data_bucket_name" {
  type        = string
  description = "Globally-unique S3 bucket name for app data."
}

variable "ses_domain" {
  type        = string
  description = "Domain (or subdomain) that receives mail via SES, e.g. inbox.example.com."
}

variable "route53_zone_id" {
  type        = string
  default     = null
  description = "Route53 zone id for ses_domain's parent zone, if managed there. Leave null to set up DNS manually."
}

variable "allowed_sender_email" {
  type        = string
  description = "Only mail From this address is processed by email-router; everything else is dropped."
}

variable "log_retention_days" {
  type    = number
  default = 14
}
