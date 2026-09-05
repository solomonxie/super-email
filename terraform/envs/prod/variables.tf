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

variable "ec2_instance_type" {
  type    = string
  default = "t3.micro"
}

variable "ec2_ssh_public_key" {
  type        = string
  description = "Contents of an SSH public key (e.g. contents of ~/.ssh/id_ed25519.pub) to allow onto the host."
}

variable "ssh_allowed_cidr" {
  type        = string
  description = "CIDR allowed to SSH into the host, e.g. \"1.2.3.4/32\"."
}

variable "webhook_subdomain" {
  type    = string
  default = "hooks"
}

variable "root_volume_size_gb" {
  type    = number
  default = 20
}
