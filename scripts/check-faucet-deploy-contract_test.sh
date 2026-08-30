#!/usr/bin/env bash
set -euo pipefail

workflow='.github/workflows/testnet-public-rpc.yml'

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[ -f "$workflow" ] || fail "$workflow is missing"

require_literal() {
  local needle="$1"
  grep -Fq "$needle" "$workflow" || fail "missing required deployment contract: $needle"
}

require_literal 'workflow_dispatch:'
require_literal 'deploy:'
require_literal "if: github.event_name == 'workflow_dispatch' && inputs.deploy == true"
require_literal 'uses: actions/download-artifact@v4'
require_literal 'FAUCET_ENABLED=false'
require_literal '/v1/faucet/info'
require_literal '/v1/faucet/health'
require_literal "trap 'disable_faucet' ERR"
require_literal 'trap - ERR'

if grep -Fq "if: github.ref == 'refs/heads/feature/public-testnet-wallet-v2'" "$workflow"; then
  fail 'deploy remains hard-wired to the historical feature/public-testnet-wallet-v2 branch'
fi

stage_line="$(grep -n -m1 'name: Stage faucet disabled' "$workflow" | cut -d: -f1 || true)"
code_line="$(grep -n -m1 'name: Update Lambda code from tested artifact' "$workflow" | cut -d: -f1 || true)"
[ -n "$stage_line" ] || fail 'missing Stage faucet disabled step'
[ -n "$code_line" ] || fail 'missing Update Lambda code from tested artifact step'
if [ "$stage_line" -ge "$code_line" ]; then
  fail 'faucet must be forced disabled before new Lambda code is installed'
fi

printf 'PASS: faucet deployment contract is manual-only, deep-health gated, fail-closed on unexpected errors, disables before code update, and promotes the tested artifact\n'
