#!/usr/bin/env bash
set -euo pipefail

workflow='.github/workflows/demand-miner-auto-deploy.yml'
monitor='.github/workflows/faucet-recovery-monitor.yml'

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

require_literal() {
  local file="$1"
  local needle="$2"
  grep -Fq -- "$needle" "$file" || fail "missing in $file: $needle"
}

[ -f "$workflow" ] || fail "$workflow is missing"
[ -f "$monitor" ] || fail "$monitor is missing"
[ -f deployment/testnet/remote-install-demand-miner-from-urls.sh ] || fail 'remote install-from-urls script is missing'
[ -f scripts/publish-demand-miner-binaries.sh ] || fail 'publish-demand-miner-binaries script is missing'

require_literal "$workflow" 'name: Demand Miner Auto Deploy'
require_literal "$workflow" 'workflow_call:'
require_literal "$workflow" 'assess-chain-work:'
require_literal "$workflow" 'ensure-on-seed:'
require_literal "$workflow" 'publish-demand-miner-binaries.sh'
require_literal "$workflow" 'remote-install-demand-miner-from-urls.sh'
require_literal "$workflow" 'on-seed build fallback'
require_literal "$workflow" 'base64 -d | bash'
require_literal "$workflow" 'aws ssm send-command'
require_literal "$workflow" 'trigger-faucet-recovery:'
require_literal "$monitor" 'auto-deploy-demand-miner:'
require_literal "$monitor" './.github/workflows/demand-miner-auto-deploy.yml'

printf 'PASS: demand miner auto-deploy workflow is wired for automatic seed ensure\n'
