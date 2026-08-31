#!/usr/bin/env bash
set -euo pipefail

if (( $# < 2 )); then
  printf 'usage: %s VPC_ID SUBNET_ID...\n' "$0" >&2
  exit 64
fi

vpc_id="$1"
shift
aws_bin="${AWS_BIN:-aws}"
declare -A seen=()

for subnet_id in "$@"; do
  route_table_id="$($aws_bin ec2 describe-route-tables \
    --filters "Name=association.subnet-id,Values=${subnet_id}" \
    --query 'RouteTables[0].RouteTableId' --output text)"

  if [[ -z "$route_table_id" || "$route_table_id" == 'None' ]]; then
    route_table_id="$($aws_bin ec2 describe-route-tables \
      --filters 'Name=association.main,Values=true' "Name=vpc-id,Values=${vpc_id}" \
      --query 'RouteTables[0].RouteTableId' --output text)"
  fi

  if [[ -z "$route_table_id" || "$route_table_id" == 'None' ]]; then
    printf 'no effective route table found for subnet %s\n' "$subnet_id" >&2
    exit 1
  fi

  if [[ -z "${seen[$route_table_id]:-}" ]]; then
    printf '%s\n' "$route_table_id"
    seen[$route_table_id]=1
  fi
done
