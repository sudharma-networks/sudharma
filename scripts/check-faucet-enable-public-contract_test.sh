#!/usr/bin/env bash
set -euo pipefail

workflow='.github/workflows/faucet-enable-public.yml'

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[ -f "$workflow" ] || fail "$workflow is missing"
[ -f scripts/faucet-live-e2e.mjs ] || fail 'missing live faucet e2e script'
[ -f scripts/faucet-fee-utils.mjs ] || fail 'missing faucet fee utils'

require_literal() {
  grep -Fq -- "$1" "$workflow" || fail "missing public faucet enable contract: $1"
}

require_literal 'name: Faucet Enable Public'
require_literal 'feature/faucet-recovery-stage2'
require_literal 'FAUCET_ENABLED: '\''true'\'''
require_literal 'node ./scripts/faucet-live-e2e.mjs'
require_literal 'Confirm faucet remains enabled'
require_literal 'development_fee'
require_literal 'treasury_increase'

if grep -Fq -- 'Disable faucet again' "$workflow"; then
  fail 'public faucet enable must not disable faucet after verification'
fi

printf 'PASS: public faucet enable deploys live code, keeps faucet enabled, and runs treasury-checked e2e\n'
