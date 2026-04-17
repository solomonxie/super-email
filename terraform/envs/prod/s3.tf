# Depends on: variables.tf (var.data_bucket_name).
# Depended on by: lambdas.tf (bucket name/arn for IAM statements + env
#   vars), ses.tf (bucket name/arn for the receipt rule + bucket policy).

module "data_bucket" {
  source      = "../../modules/s3-data"
  bucket_name = var.data_bucket_name
}
