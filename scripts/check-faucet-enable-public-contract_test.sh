#!/usr/bin/env bash
set -euo pipefail

workflow='.github/workflows/faucet-enable-public.yml'

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[ -f "$workflow" ] || fail "$workflow is missing"
[ -f scripts/faucet-live-e2e.mjs ] || fail 'missing live faucet e2e script'
[ -f scripts/explorer-live-check.mjs ] || fail 'missing explorer live check script'

require_literal() {
  grep -Fq -- "$1" "$workflow" || fail "missing public faucet enable contract: $1"
}

require_literal 'name: Faucet Enable Public'
require_literal 'workflow_dispatch:'
require_literal 'FAUCET_ENABLED: '\''true'\'''
require_literal 'node ./scripts/faucet-live-e2e.mjs'
require_literal 'node ./scripts/explorer-live-check.mjs'
require_literal 'faucet-funding.mjs'
require_literal 'miner-wake.mjs'
require_literal 'Confirm faucet remains enabled'
require_literal 'development_fee'
require_literal 'treasury_increase'

if grep -Fq -- 'Disable faucet again' "$workflow"; then
  fail 'public faucet enable must not disable faucet after verification'
fi

if grep -Eq '^[[:space:]]+push:' "$workflow"; then
  fail 'public faucet enable must be manual-only and must not deploy on branch pushes'
fi

grep -Fq -- 'fix/faucet-health-timeout' .github/workflows/faucet-recovery-ci.yml \
  || fail 'temporary faucet branch must run non-deploying recovery CI'

printf 'PASS: public faucet enable deploys live code, verifies explorer, and keeps faucet enabled\n'
