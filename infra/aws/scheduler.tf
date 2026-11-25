# Reliable clock for the dashboard: an EventBridge rule with a cron schedule
# fires every 15 minutes (07:00-22:45 BRT) targeting an API destination that
# dispatches the GitHub Actions workflow. GitHub's own cron scheduler was
# delivering ~3 runs/day, so the clock lives here instead.

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

# Classic EventBridge rules evaluate cron in UTC only. This window is
# UTC 10:00-23:59 + 00:00-01:45 = 07:00-22:45 America/Sao_Paulo.
resource "aws_cloudwatch_event_rule" "render" {
  name                = "kindle-dashboard-render"
  description         = "Triggers the render workflow during daytime hours (BRT)"
  schedule_expression = var.schedule_expression
}

resource "aws_cloudwatch_event_target" "render" {
  rule     = aws_cloudwatch_event_rule.render.name
  arn      = aws_cloudwatch_event_api_destination.github.arn
  role_arn = aws_iam_role.render_rule.arn
  input    = jsonencode({ ref = "main" })
}

data "aws_iam_policy_document" "render_rule_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["events.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "render_rule" {
  name               = "kindle-dashboard-render-rule"
  assume_role_policy = data.aws_iam_policy_document.render_rule_assume.json
}

data "aws_iam_policy_document" "render_rule" {
  statement {
    actions   = ["events:InvokeApiDestination"]
    resources = [aws_cloudwatch_event_api_destination.github.arn]
  }
}

resource "aws_iam_role_policy" "render_rule" {
  name   = "invoke-dispatch"
  role   = aws_iam_role.render_rule.id
  policy = data.aws_iam_policy_document.render_rule.json
}
