# Depends on: s3.tf (bucket arn, for the instance role's IAM policy),
#   variables.tf.
# Depended on by: ses.tf (webhook_fqdn → webhook_url), outputs.tf.

module "app_host" {
  source = "../../modules/ec2-host"

  project_name        = var.project_name
  instance_type       = var.ec2_instance_type
  root_volume_size_gb = var.root_volume_size_gb
  ssh_public_key      = var.ec2_ssh_public_key
  ssh_allowed_cidr    = var.ssh_allowed_cidr
  data_bucket_arn     = module.data_bucket.bucket_arn
  domain_name         = var.ses_domain
  webhook_subdomain   = var.webhook_subdomain
  route53_zone_id     = var.route53_zone_id
}
