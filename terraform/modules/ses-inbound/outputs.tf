output "verification_token" {
  value = aws_ses_domain_identity.this.verification_token
}

output "dkim_tokens" {
  value = aws_ses_domain_dkim.this.dkim_tokens
}

output "mail_from_domain" {
  value = aws_ses_domain_mail_from.this.mail_from_domain
}

output "receipt_rule_set_name" {
  value = aws_ses_receipt_rule_set.this.rule_set_name
}
