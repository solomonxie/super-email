# Depends on: nothing.
# Depended on by: lambdas.tf, ses.tf (account_id for IAM/permission resources).

data "aws_caller_identity" "current" {}
