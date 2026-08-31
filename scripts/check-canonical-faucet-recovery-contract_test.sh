#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

require_literal() {
  grep -Fq -- "$1" "$2" || fail "missing in $2: $1"
}

for file in \
  deployment/testnet/public-rpc/lambda/faucet-health-route.test.mjs \
  deployment/testnet/public-rpc/lambda/faucet-stale-prepared.test.mjs \
  deployment/testnet/public-rpc/lambda/faucet-funding.mjs \
  deployment/testnet/public-rpc/lambda/miner-wake.mjs \
  demandminer/faucet_funding.go \
  demandminer/wake.go; do
  [ -f "$file" ] || fail "$file is missing"
done

require_literal "kind: 'faucetHealth'" deployment/testnet/public-rpc/lambda/router.mjs
require_literal 'wakeDemandMinerInBackground' deployment/testnet/public-rpc/lambda/index.mjs
require_literal 'waitForFaucetFunding' deployment/testnet/public-rpc/lambda/faucet.mjs
require_literal 'mineFaucetFundingBlocks' demandminer/faucet_funding.go
require_literal 'NewWakeServer' demandminer/wake.go
require_literal 'checkFaucetReadiness' deployment/testnet/public-rpc/lambda/faucet-runtime.mjs
require_literal 'faucet dependency timed out' deployment/testnet/public-rpc/lambda/faucet-runtime.mjs

workflow='.github/workflows/testnet-public-rpc.yml'
[ -f "$workflow" ] || fail "$workflow is missing"
require_literal 'workflow_dispatch:' "$workflow"
require_literal 'faucet-funding.mjs' "$workflow"
require_literal 'miner-wake.mjs' "$workflow"
require_literal '/v1/faucet/health' "$workflow"

if grep -Eq '^[[:space:]]+push:' "$workflow"; then
  fail 'testnet public rpc workflow must be manual-only at the root trigger'
fi

if grep -Fq -- "refs/heads/feature/public-testnet-wallet-v2" "$workflow"; then
  fail 'testnet public rpc deploy must not be gated to legacy wallet branch'
fi

printf 'PASS: canonical faucet recovery contract is present\n'
