#!/usr/bin/env bash
# Deterministic mainnet monetary rehearsal for operator review.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

echo "Running mainnet monetary rehearsal tests..."
go test ./consensus -run 'TestMainnet' -count=1
go test ./blockchain -run 'TestMainnetMonetaryRehearsalSample|TestProcessBlockForMainnet|TestCreditMinerRewardForMainnetUsesMainnetSubsidyAndMinerFees|TestMintSupplyForMainnetEnforcesMainnetCap' -count=1

echo "Publishing mainnet genesis candidate info..."
go run ./cmd/sudharma-mainnet-genesis-info | tee /tmp/sudharma-mainnet-genesis-info.json

echo "Publishing mainnet readiness gates..."
go run ./cmd/sudharma-mainnet-readiness | tee /tmp/sudharma-mainnet-readiness.json

jq -e '.launch_ready == false' /tmp/sudharma-mainnet-readiness.json >/dev/null
jq -e '.launch_authorized == false and .mining_authorized == false' /tmp/sudharma-mainnet-genesis-info.json >/dev/null

echo '{"mainnet_monetary_rehearsal":"ok"}'
