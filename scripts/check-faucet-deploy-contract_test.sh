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

if grep -Fq "if: github.ref == 'refs/heads/feature/public-testnet-wallet-v2'" "$workflow"; then
  fail 'deploy remains hard-wired to the historical feature/public-testnet-wallet-v2 branch'
fi

printf 'PASS: faucet deployment contract is manual-only and promotes the tested artifact\n'
