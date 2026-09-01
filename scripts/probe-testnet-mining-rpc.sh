#!/usr/bin/env bash
# Probe public-testnet GPU mining RPC for solo miners and pool operators.
set -euo pipefail

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

rpc_url="${MINING_RPC_URL:-https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com}"
address="${POOL_PAYOUT_ADDRESS:-9ccdc094489874bed888ffe4bdf9b8298f4c5131}"

if ! [[ "$address" =~ ^[0-9a-f]{40}$ ]]; then
  fail "POOL_PAYOUT_ADDRESS must be 40 lowercase hex chars, got: $address"
fi

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

http_code="$(curl -fsS -o "$tmp" -w '%{http_code}' -X POST "${rpc_url%/}/v1/mining/work" \
  -H 'content-type: application/json' \
  --data "{\"address\":\"${address}\"}")"

if [[ "$http_code" != "200" ]]; then
  fail "mining work returned HTTP $http_code from $rpc_url"
fi

algorithm="$(jq -er '.algorithm' "$tmp")"
height="$(jq -er '.height' "$tmp")"
job="$(jq -er '.job' "$tmp")"

if [[ "$algorithm" != "sudharma-gpupow-v1" ]]; then
  fail "unexpected algorithm: $algorithm"
fi
if [[ "$job" != "candidate-block" ]]; then
  fail "unexpected job type: $job"
fi
if [[ "$height" =~ ^[0-9]+$ ]] && (( height < 1 )); then
  fail "unexpected block height: $height"
fi

printf 'PASS: mining RPC ready at %s (height=%s algorithm=%s)\n' "$rpc_url" "$height" "$algorithm"
