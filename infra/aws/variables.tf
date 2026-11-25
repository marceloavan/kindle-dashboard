variable "github_repo" {
  description = "Repository whose workflow is dispatched."
  type        = string
  default     = "marceloavan/kindle-dashboard"
}

variable "workflow_file" {
  description = "Workflow file to dispatch."
  type        = string
  default     = "render.yml"
}

variable "github_token" {
  description = "Fine-grained GitHub PAT with actions:write on the repo. Pass via TF_VAR_github_token; never commit."
  type        = string
  sensitive   = true
}

variable "schedule_expression" {
  description = "Cron for the EventBridge rule (6 fields incl. year; UTC only)."
  type        = string
  # UTC 10:00-23:59 + 00:00-01:45 = 07:00-22:45 America/Sao_Paulo.
  default = "cron(*/15 10-23,0-1 * * ? *)"
}
