# Depends on: s3-data (bucket arn, for the instance role's IAM policy).
# Depended on by: envs/prod (public_ip/webhook_fqdn feed ses-inbound's
#   webhook_url and the manual DNS/SSH steps in outputs.tf).
#
# aws_security_group.this   (22 from ssh_allowed_cidr, 80/443 open for
#                             ACME + the inbound-webhook)
# aws_key_pair.this         (from ssh_public_key, no local keypair mgmt)
# aws_iam_role.this ──┬─→ aws_iam_role_policy.s3   (bucket read/write)
#                     └─→ aws_iam_role_policy.ses  (SendEmail/SendRawEmail)
# aws_instance.this         (default VPC/subnet, user_data installs
#                             Docker + Compose plugin + Caddy only —
#                             app deploy itself is a manual ssh step)
# aws_eip.this               → aws_route53_record.webhook (only if
#                              route53_zone_id set)
#
# DESIGN.md §9: Terraform provisions this box; `docker compose up -d`
# over SSH is the app deploy, same as it would be on a hand-made VPS.

data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

data "aws_ami" "al2023" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-*-x86_64"]
  }
}

resource "aws_security_group" "this" {
  name        = "${var.project_name}-host"
  description = "SSH from an allow-listed CIDR; 80/443 open for ACME + the inbound-webhook."
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.ssh_allowed_cidr]
  }

  ingress {
    description = "HTTP (ACME challenge, redirects to HTTPS)"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "HTTPS (inbound-webhook)"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_key_pair" "this" {
  key_name   = "${var.project_name}-host"
  public_key = var.ssh_public_key
}

resource "aws_iam_role" "this" {
  name = "${var.project_name}-host"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy" "s3" {
  name = "data-bucket"
  role = aws_iam_role.this.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["s3:GetObject", "s3:PutObject", "s3:ListBucket"]
      Resource = [var.data_bucket_arn, "${var.data_bucket_arn}/*"]
    }]
  })
}

resource "aws_iam_role_policy" "ses" {
  name = "send-mail"
  role = aws_iam_role.this.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["ses:SendEmail", "ses:SendRawEmail"]
      Resource = "*"
    }]
  })
}

resource "aws_iam_instance_profile" "this" {
  name = "${var.project_name}-host"
  role = aws_iam_role.this.name
}

# Installs the runtime only — no repo checkout, no `docker compose up`.
# Getting the app onto the box is a manual/CI ssh step (terraform/README.md).
locals {
  user_data = <<-EOF
    #!/bin/bash
    set -eux
    dnf install -y docker
    systemctl enable --now docker
    usermod -aG docker ec2-user

    mkdir -p /usr/local/lib/docker/cli-plugins
    curl -fsSL "https://github.com/docker/compose/releases/latest/download/docker-compose-linux-x86_64" \
      -o /usr/local/lib/docker/cli-plugins/docker-compose
    chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

    dnf install -y 'dnf-command(copr)'
    dnf copr enable -y @caddy/caddy
    dnf install -y caddy

    mkdir -p /opt/${var.project_name}
  EOF
}

resource "aws_instance" "this" {
  ami                    = data.aws_ami.al2023.id
  instance_type          = var.instance_type
  subnet_id              = data.aws_subnets.default.ids[0]
  vpc_security_group_ids = [aws_security_group.this.id]
  iam_instance_profile   = aws_iam_instance_profile.this.name
  key_name               = aws_key_pair.this.key_name
  user_data              = local.user_data

  root_block_device {
    volume_type = "gp3"
    volume_size = var.root_volume_size_gb
  }

  tags = {
    Name = "${var.project_name}-host"
  }
}

resource "aws_eip" "this" {
  instance = aws_instance.this.id
  domain   = "vpc"
}

resource "aws_route53_record" "webhook" {
  count   = var.route53_zone_id == null ? 0 : 1
  zone_id = var.route53_zone_id
  name    = "${var.webhook_subdomain}.${var.domain_name}"
  type    = "A"
  ttl     = 300
  records = [aws_eip.this.public_ip]
}
