#!/usr/bin/env bash
# Builds linux/amd64 demand miner binaries and uploads them to S3.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bucket="${TESTNET_DEMAND_MINER_BUCKET:-sudharma-testnet-demand-miner-981626123397}"
prefix="${DEMAND_MINER_S3_PREFIX:-demand-miner/${GITHUB_SHA:-manual}}"
region="${AWS_REGION:-ap-south-1}"
out_dir="$(mktemp -d)"

cleanup() { rm -rf "$out_dir"; }
trap cleanup EXIT

(
  cd "$repo_root"
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -trimpath -o "$out_dir/sudharma-demand-miner" ./cmd/sudharma-demand-miner
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -trimpath -o "$out_dir/sudharmad" ./cmd/sudharmad
)

aws s3api head-bucket --bucket "$bucket" >/dev/null 2>&1 || {
  echo "Creating S3 bucket s3://$bucket in $region ..."
  if [ "$region" = "us-east-1" ]; then
    aws s3api create-bucket --bucket "$bucket" --region "$region"
  else
    aws s3api create-bucket --bucket "$bucket" --region "$region" \
      --create-bucket-configuration "LocationConstraint=$region"
  fi
  aws s3api put-public-access-block --bucket "$bucket" \
    --public-access-block-configuration \
    BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
}

aws s3api head-bucket --bucket "$bucket" >/dev/null 2>&1 || {
  echo "S3 bucket s3://$bucket does not exist or is not accessible" >&2
  exit 2
}

aws s3 cp "$out_dir/sudharma-demand-miner" "s3://${bucket}/${prefix}/sudharma-demand-miner" --region "$region"
aws s3 cp "$out_dir/sudharmad" "s3://${bucket}/${prefix}/sudharmad" --region "$region"

demand_url="$(aws s3 presign "s3://${bucket}/${prefix}/sudharma-demand-miner" --expires-in 900 --region "$region")"
node_url="$(aws s3 presign "s3://${bucket}/${prefix}/sudharmad" --expires-in 900 --region "$region")"

jq -nc \
  --arg bucket "$bucket" \
  --arg prefix "$prefix" \
  --arg demand_url "$demand_url" \
  --arg node_url "$node_url" \
  '{bucket:$bucket,prefix:$prefix,demand_miner_bin_url:$demand_url,sudharmad_bin_url:$node_url}'
