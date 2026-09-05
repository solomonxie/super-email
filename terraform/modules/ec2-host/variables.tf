variable "project_name" {
  type = string
}

variable "instance_type" {
  type    = string
  default = "t3.micro"
}

variable "root_volume_size_gb" {
  type    = number
  default = 20
}

variable "ssh_public_key" {
  type        = string
  description = "Contents of an SSH public key (e.g. ~/.ssh/id_ed25519.pub) to allow onto the box."
}

variable "ssh_allowed_cidr" {
  type        = string
  description = "CIDR allowed to SSH in, e.g. \"1.2.3.4/32\"."
}

variable "data_bucket_arn" {
  type        = string
  description = "ARN of the S3 data bucket the instance role may read/write."
}

variable "domain_name" {
  type        = string
  description = "Domain that receives mail via SES; the webhook subdomain is carved from it."
}

variable "webhook_subdomain" {
  type    = string
  default = "hooks"
}

variable "route53_zone_id" {
  type        = string
  default     = null
  description = "Route53 zone id for domain_name's parent zone. If null, point the webhook subdomain at the output public_ip manually."
}
