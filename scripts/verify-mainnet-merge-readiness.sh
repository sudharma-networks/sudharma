#!/usr/bin/env bash
# One-shot verification for mainnet merge reviewers and operators.
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: verify-mainnet-merge-readiness.sh [pr76|pr77|all|parallel]

  pr76      Tokenomics-only checks (PR #76 reviewers)
  pr77      Readiness + mining stack checks (PR #77 reviewers, default)
  all       pr76 then pr77
  parallel  Live testnet mining RPC probe + pool smoke (needs network)

Environment:
  SKIP_LIVE_PROBE=1   Skip curl probe in parallel mode
EOF
}

mode="${1:-pr77}"

case "$mode" in
  pr76|pr77|all|parallel) ;;
  -h|--help) usage; exit 0 ;;
  *) echo "unknown mode: $mode" >&2; usage >&2; exit 2 ;;
esac

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

run_pr76() {
  echo "== PR #76 tokenomics verification =="
  go vet ./...
  go test ./consensus -run TestMainnet -count=1
  go test ./blockchain -run 'TestProcessBlockForMainnet|TestMintSupplyForMainnetEnforcesMainnetCap|TestCreditMinerRewardForMainnet' -count=1
  go test ./params -count=1
  echo '{"pr76_verification":"ok"}'
}

run_pr77() {
  echo "== PR #77 mainnet readiness verification =="
  go test ./... -count=1
  go run ./cmd/sudharma-mainnet-readiness >/tmp/sudharma-mainnet-readiness.json
  go run ./cmd/sudharma-mining-readiness >/tmp/sudharma-mining-readiness.json
  go run ./cmd/sudharma-mainnet-genesis-info >/tmp/sudharma-mainnet-genesis-info.json
  bash ./scripts/mainnet-monetary-rehearsal.sh
  bash ./scripts/check-mainnet-readiness-contract_test.sh
  bash ./scripts/check-mainnet-go-live-readiness_test.sh
  bash ./scripts/check-mainnet-merge-review-contract_test.sh
  bash ./scripts/check-pool-mining-contract_test.sh
  bash ./scripts/check-mining-readiness-contract_test.sh
  bash ./scripts/pool-mining-smoke_test.sh
  jq -e '.launch_ready == false and .launch_authorized == false and .mining_stack_ready == true' /tmp/sudharma-mainnet-readiness.json >/dev/null
  jq -e '.stack_ready == true and .mainnet_mining_authorized == false' /tmp/sudharma-mining-readiness.json >/dev/null
  jq -e '.launch_authorized == false and .mining_authorized == false' /tmp/sudharma-mainnet-genesis-info.json >/dev/null
  echo '{"pr77_verification":"ok"}'
}

run_parallel() {
  echo "== Parallel operator verification (testnet surfaces) =="
  bash ./scripts/pool-mining-smoke_test.sh
  if [[ "${SKIP_LIVE_PROBE:-}" == "1" ]]; then
    echo "SKIP_LIVE_PROBE=1 — skipping live mining RPC probe"
  else
    bash ./scripts/probe-testnet-mining-rpc.sh
  fi
  echo '{"parallel_verification":"ok"}'
}

case "$mode" in
  pr76) run_pr76 ;;
  pr77) run_pr77 ;;
  all) run_pr76; run_pr77 ;;
  parallel) run_parallel ;;
esac
