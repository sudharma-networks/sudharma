#!/usr/bin/env bash
# Engineering self-check before external security audit (does NOT close audit gate).
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

failures=0
pass() { printf 'PASS: %s\n' "$1"; }
fail() { printf 'FAIL: %s\n' "$1" >&2; failures=$((failures + 1)); }

echo "Sudharma pre-audit engineering self-check"
echo "Repository: $(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
echo "Note: this does not substitute for an independent security audit."
echo

run() {
  if "$@"; then
    pass "$*"
  else
    fail "$*"
  fi
}

run go vet ./...
run go test ./params ./consensus ./blockchain ./p2p ./rpc ./pool ./gpuminer/... ./cmd/sudharmad ./cmd/sudharma-miner ./cmd/sudharma-pool -count=1
run bash ./scripts/check-tracked-secrets_test.sh
run bash ./scripts/check-mainnet-readiness-contract_test.sh
run bash ./scripts/check-mainnet-go-live-readiness_test.sh
run bash ./scripts/check-mainnet-merge-review-contract_test.sh
run bash ./scripts/check-mining-readiness-contract_test.sh
run bash ./scripts/check-pool-mining-contract_test.sh
run bash ./scripts/check-mining-api-contract_test.sh
run bash ./scripts/pool-mining-smoke_test.sh
run bash ./scripts/mainnet-monetary-rehearsal.sh

if ! go run ./cmd/sudharma-mainnet-readiness | jq -e '.launch_authorized == false and .launch_ready == false and .mining_stack_ready == true' >/dev/null; then
  fail 'mainnet readiness JSON gates'
else
  pass 'mainnet readiness JSON gates'
fi

if ! go run ./cmd/sudharma-mining-readiness | jq -e '.stack_ready == true and .mainnet_mining_authorized == false' >/dev/null; then
  fail 'mining readiness JSON gates'
else
  pass 'mining readiness JSON gates'
fi

if grep -R --line-number 'MainnetLaunchAuthorized = true' params cmd blockchain 2>/dev/null; then
  fail 'forbidden MainnetLaunchAuthorized = true in tree'
else
  pass 'MainnetLaunchAuthorized stays false in tree'
fi

if [[ "${SKIP_LIVE_PROBE:-}" == "1" ]]; then
  echo "SKIP_LIVE_PROBE=1 — skipping live testnet mining RPC probe"
else
  run bash ./scripts/probe-testnet-mining-rpc.sh
fi

echo
if (( failures > 0 )); then
  echo "{\"pre_audit_selfcheck\":\"failed\",\"failures\":$failures}"
  exit 1
fi

echo '{"pre_audit_selfcheck":"ok"}'
