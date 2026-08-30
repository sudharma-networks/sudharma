#!/usr/bin/env bash
set -euo pipefail

script="deployment/testnet/rollout-explorer-node.sh"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

[[ -f "$script" ]] || fail "$script is missing"

require_literal() {
  local text="$1"
  grep -Fq -- "$text" "$script" || fail "missing safety contract: $text"
}

require_literal 'EXPECTED_OLD_SHA'
require_literal 'EXPECTED_NEW_SHA'
require_literal 'EXPECTED_NODE_ID'
require_literal 'ARTIFACT_URL_B64'
require_literal 'sha256sum'
require_literal 'sudharma-rpcd.rollback-'
require_literal 'systemctl restart sudharma.service'
require_literal 'http://127.0.0.1:28545/ready'
require_literal 'http://127.0.0.1:28545/v1/status'
require_literal 'http://127.0.0.1:28545/v1/explorer/status'
require_literal '29100/v1/explorer/status'
require_literal 'before_height'
require_literal 'before_supply'
require_literal 'peers'
require_literal 'Rolling back node binary.'

if grep -Eiq '(cuda|opencl|gpu-pow|khushi|systemctl[[:space:]]+(enable|start|restart)[[:space:]]+.*miner|ufw|iptables|nft|/etc/sudharma/node\.json.*>|sed .*node\.json)' "$script"; then
  fail 'rollout script contains a prohibited GPU/miner/firewall/config mutation'
fi

restart_count="$(grep -F 'systemctl restart sudharma.service' "$script" | wc -l | tr -d ' ')"
[[ "$restart_count" -ge 2 ]] || fail 'expected restart path plus rollback restart path'

require_literal "current_sha=\"\$(sha256sum /usr/local/bin/sudharma-rpcd"
require_literal "installed_sha=\"\$(sha256sum /usr/local/bin/sudharma-rpcd"
require_literal 'fail_after_change'

printf 'PASS: explorer seed rollout safety contract\n'
