#!/usr/bin/env bash
# Builds linux/amd64 sudharma-pool and uploads it to S3 for operator installs.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bucket="${TESTNET_DEMAND_MINER_BUCKET:-sudharma-testnet-demand-miner-981626123397}"
prefix="${SUDHARMA_POOL_S3_PREFIX:-sudharma-pool/${GITHUB_SHA:-manual}}"
region="${AWS_REGION:-ap-south-1}"
out_dir="$(mktemp -d)"

cleanup() { rm -rf "$out_dir"; }
trap cleanup EXIT

(
  cd "$repo_root"
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w" -o "$out_dir/sudharma-pool" ./cmd/sudharma-pool
)

aws s3api head-bucket --bucket "$bucket" >/dev/null 2>&1 || {
  echo "S3 bucket s3://$bucket does not exist or is not accessible" >&2
  exit 2
}

aws s3 cp "$out_dir/sudharma-pool" "s3://${bucket}/${prefix}/sudharma-pool" \
  --only-show-errors --region "$region"
pool_url="$(aws s3 presign "s3://${bucket}/${prefix}/sudharma-pool" --expires-in 900 --region "$region")"

jq -nc \
  --arg bucket "$bucket" \
  --arg prefix "$prefix" \
  --arg pool_url "$pool_url" \
  '{bucket:$bucket,prefix:$prefix,sudharma_pool_bin_url:$pool_url}'
