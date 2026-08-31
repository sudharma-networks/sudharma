#!/usr/bin/env bash
set -euo pipefail

workflow='.github/workflows/faucet-prepared-payout-recovery.yml'

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[ -f "$workflow" ] || fail "$workflow is missing"

require_literal() {
  grep -Fq -- "$1" "$workflow" || fail "missing prepared payout recovery contract: $1"
}

require_literal 'name: Faucet Prepared Payout Recovery'
require_literal 'workflow_dispatch:'
require_literal 'FAILED_ADDRESS: 16d7dc9ec0495109007860a584c7cf9055da9abf'
require_literal 'Resubmit prepared payout for failed address only'
require_literal '/v1/faucet/request'

if grep -Eq '^[[:space:]]+(push|schedule|workflow_run|workflow_call):' "$workflow"; then
  fail 'prepared payout recovery must be manual-only'
fi

if grep -Fq -- 'Disable faucet again' "$workflow"; then
  fail 'prepared payout recovery must not disable faucet after brief enable while public faucet is live'
fi

if grep -Fq -- "FAUCET_ENABLED: 'true'" "$workflow" && ! grep -Fq -- 'lambda-environment-enabled.json' "$workflow"; then
  fail 'recovery must only enable faucet through the controlled enable snapshot'
fi

printf 'PASS: prepared payout recovery stays single-address and does not disable public faucet\n'
