# Depends on: lambdas.tf (module.lambdas["digest-*"]).
# Depended on by: outputs.tf.
#
# Times are UTC, staggered an hour apart so digests don't all fire at
# once; adjust to taste once real send times matter.

module "digest_schedules" {
  source = "../../modules/scheduler"

  project_name = var.project_name

  schedules = {
    bible = {
      lambda_arn = module.lambdas["digest-bible"].function_arn
      expression = "cron(0 12 * * ? *)"
      input      = jsonencode({ digest = "bible" })
    }
    youtube = {
      lambda_arn = module.lambdas["digest-youtube"].function_arn
      expression = "cron(0 13 * * ? *)"
      input      = jsonencode({ digest = "youtube" })
    }
    social = {
      lambda_arn = module.lambdas["digest-social"].function_arn
      expression = "cron(0 14 * * ? *)"
      input      = jsonencode({ digest = "social" })
    }
    substack = {
      lambda_arn = module.lambdas["digest-substack"].function_arn
      expression = "cron(0 15 * * ? *)"
      input      = jsonencode({ digest = "substack" })
    }
  }
}
