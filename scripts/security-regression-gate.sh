#!/usr/bin/env bash
# Full security regression, race and adversarial gate for mainnet readiness.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

echo "Sudharma security regression/race/adversarial gate"
echo "Repository: $(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
echo

run() {
  printf '== %s ==\n' "$*"
  "$@"
}

run go vet ./...

run go test ./params ./consensus ./blockchain ./blockchain/mempool ./transactions ./wallet ./p2p ./rpc ./pool ./gpuminer/... \
  -run 'TestCrossNetwork|TestLegacySignature|TestMainnetRequires|TestValidateResourceBounds|TestMempoolRejects|TestMempoolCached|TestMempoolRemoval|TestTransactionsForSender|TestLocalSubmission|TestSequentialDustSpam|TestDecodeTransactionForNetwork|TestNetworkBound|TestReorg|TestRejectCrossNetwork|TestLoadChainFromFileFor|TestSetLocalNetworkID|TestApplyTransactionCreditFailure|TestFee|TestMainnet' \
  -count=1

run go test ./... -count=1

run go test -race ./params ./consensus ./blockchain ./blockchain/mempool ./transactions ./wallet ./p2p ./rpc ./pool ./miner ./gpuminer/... ./cmd/sudharmad ./cmd/sudharma-wallet -count=1

run bash ./scripts/mainnet-monetary-rehearsal.sh
SKIP_LIVE_PROBE=1 run bash ./scripts/pre-audit-engineering-selfcheck.sh

if grep -R --line-number 'MainnetLaunchAuthorized = true' params cmd blockchain 2>/dev/null; then
  echo "forbidden MainnetLaunchAuthorized = true found" >&2
  exit 1
fi

if ! go run ./cmd/sudharma-mainnet-readiness | jq -e '.launch_authorized == false and .launch_ready == false' >/dev/null; then
  echo "mainnet readiness must remain fail-closed" >&2
  exit 1
fi

echo
echo '{"security_regression_gate":"ok"}'
