# Depends on: s3-data (bucket name/arn) and ec2-host (webhook_url) —
#   both passed in by envs/prod.
# Depended on by: nothing (leaf of the graph on the inbound side).
#
# aws_ses_domain_identity ──┬─→ aws_ses_domain_dkim
#                           ├─→ aws_ses_domain_mail_from
#                           └─→ aws_route53_record.* (only if route53_zone_id set)
#
# aws_s3_bucket_policy.allow_ses      (lets SES write raw MIME into the
#                                       shared data bucket's raw-inbox/ prefix)
# aws_sns_topic.inbound ──→ aws_sns_topic_subscription.webhook (https,
#                            delivers to inbound-webhook on the EC2 box)
# aws_ses_receipt_rule_set/rule       (recipients → s3_action, then sns_action)
#
# DESIGN.md: receiving lives on an owned domain via SES; sending
# (from the worker, via the EC2 instance role) is untouched — only
# inbound is AWS-native here. No Lambda in this pipeline.

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

resource "aws_sns_topic" "inbound" {
  name = "${replace(var.domain_name, ".", "-")}-inbound"
}

resource "aws_sns_topic_policy" "allow_ses" {
  arn = aws_sns_topic.inbound.arn
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "AllowSESPublish"
      Effect    = "Allow"
      Principal = { Service = "ses.amazonaws.com" }
      Action    = "SNS:Publish"
      Resource  = aws_sns_topic.inbound.arn
      Condition = {
        StringEquals = { "aws:SourceAccount" = var.account_id }
      }
    }]
  })
}

# HTTPS delivery requires inbound-webhook to be reachable and to answer
# SNS's subscription-confirmation handshake (a SubscribeURL GET) before
# this subscription shows as confirmed — see terraform/README.md.
resource "aws_sns_topic_subscription" "webhook" {
  topic_arn = aws_sns_topic.inbound.arn
  protocol  = "https"
  endpoint  = var.webhook_url
}

resource "aws_ses_receipt_rule" "router" {
  name          = "route-to-webhook"
  rule_set_name = aws_ses_receipt_rule_set.this.rule_set_name
  recipients    = var.recipients
  enabled       = true
  scan_enabled  = true

  s3_action {
    bucket_name       = var.data_bucket_name
    object_key_prefix = "raw-inbox/"
    position          = 1
  }

  sns_action {
    topic_arn = aws_sns_topic.inbound.arn
    position  = 2
  }

  depends_on = [aws_s3_bucket_policy.allow_ses, aws_sns_topic_policy.allow_ses]
}
