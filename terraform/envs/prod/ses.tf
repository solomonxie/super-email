# Depends on: s3.tf (bucket name/arn), lambdas.tf (module.lambdas["email-router"]),
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

  router_lambda_arn  = module.lambdas["email-router"].function_arn
  router_lambda_name = module.lambdas["email-router"].function_name
}
