# Reliable clock for the dashboard: EventBridge Scheduler fires every 15
# minutes (07:00-22:59 BRT) at an API Destination that dispatches the GitHub
# Actions workflow. GitHub's own cron scheduler was delivering ~3 runs/day,
# so the clock lives here instead.

resource "aws_cloudwatch_event_connection" "github" {
  name               = "kindle-dashboard-github"
  description        = "Bearer token for dispatching the kindle-dashboard workflow"
  authorization_type = "API_KEY"

  auth_parameters {
    api_key {
      key   = "Authorization"
      value = "Bearer ${var.github_token}"
    }

    invocation_http_parameters {
      header {
        key   = "Accept"
        value = "application/vnd.github+json"
      }
      header {
        key   = "X-GitHub-Api-Version"
        value = "2022-11-28"
      }
      header {
        key   = "User-Agent"
        value = "kindle-dashboard-scheduler"
      }
    }
  }
}

resource "aws_cloudwatch_event_api_destination" "github" {
  name                = "kindle-dashboard-dispatch"
  description         = "Dispatches the render workflow via the GitHub REST API"
  connection_arn      = aws_cloudwatch_event_connection.github.arn
  invocation_endpoint = "https://api.github.com/repos/${var.github_repo}/actions/workflows/${var.workflow_file}/dispatches"
  http_method         = "POST"
  invocation_rate_limit_per_second = 1
}

data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

# EventBridge Scheduler rejects the API destination ARN when it carries the
# trailing UUID, so the target ARN is built without it.
locals {
  api_destination_arn = "arn:aws:events:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:api-destination/${aws_cloudwatch_event_api_destination.github.name}"
}

data "aws_iam_policy_document" "scheduler" {
  statement {
    actions = ["events:InvokeApiDestination"]
    resources = [
      aws_cloudwatch_event_api_destination.github.arn, # full ARN (with UUID)
      local.api_destination_arn,                        # ARN form used by Scheduler
    ]
  }
}

resource "aws_iam_role" "scheduler" {
  name = "kindle-dashboard-scheduler"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "scheduler.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy" "scheduler" {
  name   = "invoke-dispatch"
  role   = aws_iam_role.scheduler.id
  policy = data.aws_iam_policy_document.scheduler.json
}

resource "aws_scheduler_schedule" "render" {
  name        = "kindle-dashboard-render"
  description = "Triggers the dashboard render workflow during daytime hours"

  schedule_expression          = var.schedule_expression
  schedule_expression_timezone = var.schedule_timezone

  flexible_time_window {
    mode = "OFF"
  }

  target {
    # API destinations are invoked through Scheduler's universal target, not
    # through the destination's own ARN (which CreateSchedule rejects).
    arn      = "arn:aws:scheduler:::aws-sdk:eventbridge:invokeApiDestination"
    role_arn = aws_iam_role.scheduler.arn
    input = jsonencode({
      ApiDestinationArn = aws_cloudwatch_event_api_destination.github.arn
      HttpParameters = {
        BodyParameters = { ref = "main" }
      }
    })
  }
}
