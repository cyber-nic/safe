#!/bin/sh
set -eu

bucket="${SAFE_S3_BUCKET:-safe-dev}"
region="${AWS_DEFAULT_REGION:-us-east-1}"

if ! awslocal s3api head-bucket --bucket "${bucket}" >/dev/null 2>&1; then
  awslocal s3api create-bucket \
    --bucket "${bucket}" \
    --region "${region}" >/dev/null
fi

awslocal s3api put-bucket-versioning \
  --bucket "${bucket}" \
  --versioning-configuration Status=Enabled >/dev/null

awslocal s3api put-bucket-encryption \
  --bucket "${bucket}" \
  --server-side-encryption-configuration '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}' >/dev/null

echo "bootstrapped LocalStack bucket: ${bucket}"
