# Depends on: nothing (leaf module).
# Depended on by: envs/prod (bucket name/arn passed to lambda-container,
# ses-inbound instances for IAM + receipt-rule wiring).
#
# aws_s3_bucket.data
#   ├─ public access block, SSE, versioning  (hardening)
#   └─ lifecycle rules: expire raw-inbox/ after 30d, prune old versions after 90d
#
# Single bucket, prefix-partitioned per DESIGN.md — no per-feature
# buckets, keeps IAM/lifecycle rules in one place. Prefixes:
# raw-inbox/ (expires below), attachments/ (no lifecycle rule — kept
# indefinitely, unlike the raw MIME copy).

resource "aws_s3_bucket" "data" {
  bucket = var.bucket_name
}

resource "aws_s3_bucket_public_access_block" "data" {
  bucket                  = aws_s3_bucket.data.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "data" {
  bucket = aws_s3_bucket.data.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# Versioning + a bounded noncurrent-version expiration: cheap insurance
# against a buggy email-router overwriting/deleting a note,
# without keeping history forever.
resource "aws_s3_bucket_versioning" "data" {
  bucket = aws_s3_bucket.data.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "data" {
  bucket = aws_s3_bucket.data.id

  rule {
    id     = "expire-raw-inbox"
    status = "Enabled"
    filter { prefix = "raw-inbox/" }
    expiration { days = 30 }
  }

  rule {
    id     = "prune-old-versions"
    status = "Enabled"
    filter {}
    noncurrent_version_expiration { noncurrent_days = 90 }
  }
}
