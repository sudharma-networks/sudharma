#!/usr/bin/env bash
set -euo pipefail

workflow='.github/workflows/faucet-diagnostics-auto-deploy.yml'

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[ -f "$workflow" ] || fail "$workflow is missing"

require_literal() {
  grep -Fq -- "$1" "$workflow" || fail "missing auto diagnostics deploy contract: $1"
}

require_literal 'name: Faucet Diagnostics Auto Deploy'
require_literal 'workflow_dispatch:'
require_literal 'recover-prepared-payout:'
require_literal 'deploy-diagnostics-only:'
require_literal 'Resubmit prepared payout for failed address only'
require_literal 'FAUCET_ENABLED: '\''false'\'''
require_literal 'body.enabled !== false'
require_literal 'name: Verify diagnostics-only deployment remains fail-closed'
require_literal '--zip-file fileb:///tmp/lambda-code-rollback.zip'

if grep -Fq -- 'name: Activate faucet' "$workflow"; then
  fail 'auto diagnostics deploy must not include faucet activation'
fi

if grep -Fq -- 'on:' "$workflow" && grep -Fq -- 'push:' "$workflow"; then
  fail 'auto diagnostics deploy must be manual-only so it does not fight public faucet enable'
fi

if grep -Fq -- 'Disable faucet again' "$workflow"; then
  fail 'auto diagnostics deploy must not disable faucet after recovery when public mode is active'
fi

printf 'PASS: auto diagnostics deploy is manual-only, fail-closed, and does not fight public enable\n'
