#!/usr/bin/env bash
# Send a base64-wrapped remote script to one EC2 instance via SSM and wait for completion.
set -euo pipefail

instance_id="${1:?instance id required}"
comment="${2:?comment required}"
remote_script_file="${3:?remote script file required}"
success_pattern="${4:-}"
max_attempts="${5:-90}"

wrapper_b64="$(base64 -w0 "$remote_script_file")"
params="$(jq -nc --arg b64 "$wrapper_b64" '{commands:["echo " + ($b64 | @sh) + " | base64 -d | bash"]}')"
command_id="$(aws ssm send-command \
  --instance-ids "$instance_id" \
  --document-name AWS-RunShellScript \
  --comment "$comment" \
  --parameters "$params" \
  --query 'Command.CommandId' \
  --output text)"

for attempt in $(seq 1 "$max_attempts"); do
  status="$(aws ssm get-command-invocation \
    --command-id "$command_id" \
    --instance-id "$instance_id" \
    --query 'Status' \
    --output text 2>/dev/null || echo Pending)"
  echo "ssm_status=$status instance=$instance_id attempt=$attempt"
  case "$status" in
    Success)
      invocation="$(aws ssm get-command-invocation \
        --command-id "$command_id" \
        --instance-id "$instance_id" \
        --query '{Status:Status,Stdout:StandardOutputContent,Stderr:StandardErrorContent}' \
        --output json)"
      printf '%s\n' "$invocation"
      if [ -n "$success_pattern" ] && ! printf '%s' "$invocation" | jq -e --arg p "$success_pattern" '.Stdout | test($p)' >/dev/null; then
        echo "SSM Success but stdout missing pattern: $success_pattern" >&2
        exit 1
      fi
      exit 0
      ;;
    Failed|Cancelled|TimedOut)
      aws ssm get-command-invocation \
        --command-id "$command_id" \
        --instance-id "$instance_id" \
        --query '{Status:Status,Stdout:StandardOutputContent,Stderr:StandardErrorContent}' \
        --output json
      exit 1
      ;;
  esac
  sleep 10
done

echo "SSM command timed out for $instance_id" >&2
exit 1
