variable "domain_name" {
  type        = string
  description = "Domain (or subdomain, e.g. inbox.example.com) that receives mail via SES."
}

variable "aws_region" {
  type = string
}

variable "account_id" {
  type = string
}

variable "route53_zone_id" {
  type        = string
  default     = null
  description = "Route53 hosted zone id for domain_name's parent zone. If null, DNS records are only emitted as outputs for manual setup at the registrar."
}

variable "recipients" {
  type        = list(string)
  description = "Full addresses this receipt rule accepts, e.g. [\"me@inbox.example.com\", \"note@inbox.example.com\"]."
}

variable "data_bucket_name" {
  type = string
}

variable "data_bucket_arn" {
  type = string
}

variable "webhook_url" {
  type        = string
  description = "HTTPS URL of the inbound-webhook endpoint the SNS notification is delivered to, e.g. https://hooks.inbox.example.com/ses-inbound."
}
