output "schedule_names" {
  value = { for k, s in aws_scheduler_schedule.this : k => s.name }
}
