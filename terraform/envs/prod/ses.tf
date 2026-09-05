# Depends on: s3.tf (bucket name/arn), ec2.tf (module.app_host.webhook_fqdn),
#   data.tf, variables.tf.
# Depended on by: outputs.tf.

module "ses_inbound" {
  source = "../../modules/ses-inbound"

  domain_name     = var.ses_domain
  aws_region      = var.aws_region
  account_id      = data.aws_caller_identity.current.account_id
  route53_zone_id = var.route53_zone_id

  recipients = [
    "me@${var.ses_domain}",
    "note@${var.ses_domain}",
  ]

  data_bucket_name = module.data_bucket.bucket_name
  data_bucket_arn  = module.data_bucket.bucket_arn

  webhook_url = "https://${module.app_host.webhook_fqdn}/ses-inbound"
}
