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
require_literal 'feature/faucet-recovery-stage2'
require_literal 'workflow_dispatch:'
require_literal 'recover-prepared-payout:'
require_literal 'deploy-diagnostics-only:'
require_literal 'Resubmit prepared payout for failed address only'
require_literal 'Disable faucet again'
require_literal 'FAUCET_ENABLED: '\''false'\'''
require_literal 'body.enabled !== false'
require_literal 'name: Verify diagnostics-only deployment remains fail-closed'
require_literal '--zip-file fileb:///tmp/lambda-code-rollback.zip'

if grep -Fq -- 'name: Activate faucet' "$workflow"; then
  fail 'auto diagnostics deploy must not include faucet activation'
fi

printf 'PASS: auto diagnostics deploy is push-triggered, fail-closed, and never enables payouts\n'
