output "public_ip" {
  value = aws_eip.this.public_ip
}

output "webhook_fqdn" {
  value = "${var.webhook_subdomain}.${var.domain_name}"
}

output "instance_id" {
  value = aws_instance.this.id
}
