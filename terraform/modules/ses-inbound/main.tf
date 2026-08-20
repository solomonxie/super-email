# Depends on: s3-data (bucket name/arn), lambda-container["email-router"]
#   (function arn/name) — both passed in by envs/prod.
# Depended on by: nothing (leaf of the graph on the inbound side).
#
# aws_ses_domain_identity ──┬─→ aws_ses_domain_dkim
#                           ├─→ aws_ses_domain_mail_from
#                           └─→ aws_route53_record.* (only if route53_zone_id set)
#
# aws_s3_bucket_policy.allow_ses      (lets SES write raw MIME into the
#                                       shared data bucket's raw-inbox/ prefix)
# aws_ses_receipt_rule_set/rule       (recipients → s3_action, then lambda_action)
# aws_lambda_permission.ses_invoke    (resource policy: SES may invoke the router)
#
# DESIGN.md: receiving lives on an owned domain via SES; sending
# (from any provider) is untouched — only inbound is AWS-native here.

resource "aws_ses_domain_identity" "this" {
  domain = var.domain_name
}

resource "aws_ses_domain_dkim" "this" {
  domain = aws_ses_domain_identity.this.domain
}

resource "aws_ses_domain_mail_from" "this" {
  domain           = aws_ses_domain_identity.this.domain
  mail_from_domain = "mail.${var.domain_name}"
}

# --- Optional automatic DNS (only when the zone is Route53-managed) ---

resource "aws_route53_record" "verification" {
  count   = var.route53_zone_id == null ? 0 : 1
  zone_id = var.route53_zone_id
  name    = "_amazonses.${var.domain_name}"
  type    = "TXT"
  ttl     = 600
  records = [aws_ses_domain_identity.this.verification_token]
}

resource "aws_route53_record" "dkim" {
  count   = var.route53_zone_id == null ? 0 : 3
  zone_id = var.route53_zone_id
  name    = "${aws_ses_domain_dkim.this.dkim_tokens[count.index]}._domainkey.${var.domain_name}"
  type    = "CNAME"
  ttl     = 600
  records = ["${aws_ses_domain_dkim.this.dkim_tokens[count.index]}.dkim.amazonses.com"]
}

resource "aws_route53_record" "mail_from_mx" {
  count   = var.route53_zone_id == null ? 0 : 1
  zone_id = var.route53_zone_id
  name    = aws_ses_domain_mail_from.this.mail_from_domain
  type    = "MX"
  ttl     = 600
  records = ["10 feedback-smtp.${var.aws_region}.amazonses.com"]
}

resource "aws_route53_record" "mail_from_spf" {
  count   = var.route53_zone_id == null ? 0 : 1
  zone_id = var.route53_zone_id
  name    = aws_ses_domain_mail_from.this.mail_from_domain
  type    = "TXT"
  ttl     = 600
  records = ["v=spf1 include:amazonses.com ~all"]
}

resource "aws_route53_record" "inbound_mx" {
  count   = var.route53_zone_id == null ? 0 : 1
  zone_id = var.route53_zone_id
  name    = var.domain_name
  type    = "MX"
  ttl     = 600
  records = ["10 inbound-smtp.${var.aws_region}.amazonaws.com"]
}

# --- Receiving pipeline ---

resource "aws_s3_bucket_policy" "allow_ses" {
  bucket = var.data_bucket_name
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "AllowSESPuts"
      Effect    = "Allow"
      Principal = { Service = "ses.amazonaws.com" }
      Action    = "s3:PutObject"
      Resource  = "${var.data_bucket_arn}/raw-inbox/*"
      Condition = {
        StringEquals = { "aws:SourceAccount" = var.account_id }
      }
    }]
  })
}

resource "aws_ses_receipt_rule_set" "this" {
  rule_set_name = "${replace(var.domain_name, ".", "-")}-rules"
}

resource "aws_ses_active_receipt_rule_set" "this" {
  rule_set_name = aws_ses_receipt_rule_set.this.rule_set_name
}

resource "aws_ses_receipt_rule" "router" {
  name          = "route-to-lambda"
  rule_set_name = aws_ses_receipt_rule_set.this.rule_set_name
  recipients    = var.recipients
  enabled       = true
  scan_enabled  = true

  s3_action {
    bucket_name       = var.data_bucket_name
    object_key_prefix = "raw-inbox/"
    position          = 1
  }

  lambda_action {
    function_arn    = var.router_lambda_arn
    invocation_type = "Event"
    position        = 2
  }

  depends_on = [aws_s3_bucket_policy.allow_ses, aws_lambda_permission.ses_invoke]
}

resource "aws_lambda_permission" "ses_invoke" {
  statement_id   = "AllowSESInvoke"
  action         = "lambda:InvokeFunction"
  function_name  = var.router_lambda_name
  principal      = "ses.amazonaws.com"
  source_account = var.account_id
}
