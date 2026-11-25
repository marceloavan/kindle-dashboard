output "schedule_arn" {
  description = "ARN of the EventBridge rule."
  value       = aws_cloudwatch_event_rule.render.arn
}

output "api_destination_arn" {
  description = "ARN of the API destination."
  value       = aws_cloudwatch_event_api_destination.github.arn
}
