#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

require_file() {
  [ -f "$1" ] || fail "$1 is missing"
}

for script in \
  scripts/check-canonical-faucet-recovery-contract_test.sh \
  scripts/check-explorer-api-contract_test.sh \
  scripts/check-faucet-deploy-contract_test.sh \
  scripts/check-demand-miner-auto-deploy-contract_test.sh \
  scripts/generate-testnet-rc-attestation.sh \
  scripts/testnet-rehearsal.sh \
  scripts/testnet-deploy-preflight.sh \
  scripts/live-workflow-trigger-safety.test.mjs; do
  require_file "$script"
done

for path in \
  deployment/testnet/public-profile.example.json \
  testnet/rehearsal/node1.json \
  testnet/rehearsal/node2.json \
  deployment/testnet/public-rpc/lambda/package.json \
  web/package.json \
  mobile/android \
  cmd/sudharma-testnet-manifest \
  docs/audits/2026-08-31-testnet-rc-attestation.md; do
  [ -e "$path" ] || fail "$path is missing"
done

if ! grep -Fq 'handleExplorerStatus' rpc/server.go; then
  fail 'seed RPC must expose explorer handlers for RC candidate'
fi

if ! grep -Fq 'workflow_dispatch:' .github/workflows/testnet-public-rpc.yml; then
  fail 'public RPC deploy workflow must remain manual-only'
fi

printf 'PASS: testnet release candidate readiness contract is present\n'
