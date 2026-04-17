# Depends on: nothing — first file Terraform resolves.
# Depended on by: every resource/module in this env (implicitly, via
#   the default aws provider).

terraform {
  required_version = ">= 1.7"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "~> 2.4"
    }
  }

  # Local state until a remote backend exists. Switch to S3 once you
  # have a state bucket:
  # backend "s3" {
  #   bucket = "super-email-tfstate"
  #   key    = "prod/terraform.tfstate"
  #   region = "us-east-1"
  # }
}

provider "aws" {
  region = var.aws_region
}
