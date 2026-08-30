#!/usr/bin/env bash
# Resolve an EC2 instance id for a seed host private IP.
set -euo pipefail

private_ip="${1:?private ip required}"
configured_id="${2:-}"
fallback_id="${3:-}"

if [ -n "$configured_id" ]; then
  printf '%s\n' "$configured_id"
  exit 0
fi

discovered="$(aws ssm describe-instance-information \
  --query "InstanceInformationList[?IPAddress==\`${private_ip}\`].InstanceId | [0]" \
  --output text 2>/dev/null || true)"

if [ -n "$discovered" ] && [ "$discovered" != "None" ]; then
  printf '%s\n' "$discovered"
  exit 0
fi

if [ -n "$fallback_id" ]; then
  printf '%s\n' "$fallback_id"
  exit 0
fi

exit 1
