# Depends on: s3.tf, ses.tf, scheduler.tf — prints what the next step
#   (manual DNS setup, if needed) requires.

output "data_bucket_name" {
  value = module.data_bucket.bucket_name
}

output "digest_schedule_names" {
  value = module.digest_schedules.schedule_names
}

output "manual_dns_setup_needed" {
  value       = var.route53_zone_id == null
  description = "If true, route53_zone_id wasn't set — create the records below at your registrar."
}

output "ses_verification_txt_record" {
  value       = "_amazonses.${var.ses_domain} TXT \"${module.ses_inbound.verification_token}\""
  description = "Only needed if manual_dns_setup_needed is true."
}

output "ses_dkim_cname_records" {
  value = [
    for t in module.ses_inbound.dkim_tokens :
    "${t}._domainkey.${var.ses_domain} CNAME ${t}.dkim.amazonses.com"
  ]
  description = "Only needed if manual_dns_setup_needed is true."
}

output "ses_inbound_mx_record" {
  value       = "${var.ses_domain} MX \"10 inbound-smtp.${var.aws_region}.amazonaws.com\""
  description = "Only needed if manual_dns_setup_needed is true."
}
