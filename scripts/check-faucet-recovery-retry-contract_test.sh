#!/usr/bin/env bash
set -euo pipefail

workflow='.github/workflows/faucet-recovery-retry.yml'

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[ -f "$workflow" ] || fail "$workflow is missing"

require_literal() {
  grep -Fq -- "$1" "$workflow" || fail "missing recovery retry contract: $1"
}

require_literal 'name: Faucet Recovery Retry'
require_literal 'evaluate-faucet-recovery-readiness.mjs'
require_literal 'Skip when recovery readiness says mempool still blocked'
require_literal 'Disable faucet again'
require_literal 'FAUCET_ENABLED: '\''true'\'''
require_literal 'FAUCET_ENABLED: '\''false'\'''
require_literal 'Resubmit prepared payout for failed address only'

printf 'PASS: recovery retry gates on readiness and keeps brief enable fail-closed\n'
