# Depends on: s3.tf, ec2.tf, ses.tf — prints what the next manual steps
#   (DNS, SSH, deploy) require.

output "data_bucket_name" {
  value = module.data_bucket.bucket_name
}

output "instance_public_ip" {
  value = module.app_host.public_ip
}

output "ssh_command" {
  value = "ssh ec2-user@${module.app_host.public_ip}"
}

output "webhook_url" {
  value = "https://${module.app_host.webhook_fqdn}/ses-inbound"
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

output "webhook_a_record" {
  value       = "${module.app_host.webhook_fqdn} A ${module.app_host.public_ip}"
  description = "Only needed if manual_dns_setup_needed is true."
}
