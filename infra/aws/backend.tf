terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "6.66.0"
    }
  }

  backend "s3" {
    # Bucket intentionally omitted here: it contains the AWS account ID and
    # is passed at init time via -backend-config from a masked secret
    # (TF_STATE_BUCKET) in the workflows.
    key    = "kindle-dashboard/scheduler.tfstate"
    region = "us-east-1"
  }
}

provider "aws" {
  region = "us-east-1"

  default_tags {
    tags = {
      owner = "kindle-dashboard"
    }
  }
}
