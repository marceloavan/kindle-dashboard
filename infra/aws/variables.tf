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
  description = "Cron for the scheduler, evaluated in schedule_timezone."
  type        = string
  # Every 15 minutes, 07:00-22:59 Brasília time.
  default     = "cron(*/15 7-22 * * *)"
}

variable "schedule_timezone" {
  description = "IANA timezone the cron is evaluated in."
  type        = string
  default     = "America/Sao_Paulo"
}
