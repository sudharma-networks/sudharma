#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

require_file() {
  [ -f "$1" ] || fail "$1 is missing"
}

for path in \
  params/network.go \
  params/readiness.go \
  params/mining_readiness.go \
  params/monetary.go \
  params/mainnet_emission.go \
  docs/audits/2026-08-31-mainnet-readiness.md \
  docs/audits/2026-08-31-mainnet-launch-operator-runbook.md \
  docs/audits/2026-08-31-mainnet-gpu-mining-architecture.md \
  docs/audits/2026-08-31-pool-mining-architecture.md \
  cmd/sudharma-mainnet-readiness/main.go \
  cmd/sudharma-mainnet-genesis-info/main.go; do
  require_file "$path"
done

if ! grep -Fq 'MainnetLaunchAuthorized = false' params/network.go; then
  fail 'mainnet launch must stay unauthorized in params/network.go'
fi

if ! grep -Fq 'MainnetGenesisTimestamp uint64 = 0' params/network.go; then
  fail 'mainnet genesis timestamp must remain unset until human freeze'
fi

if ! grep -Fq 'MainnetMiningAuthorized = false' params/mining.go; then
  fail 'mainnet GPU mining must stay unauthorized until launch'
fi

if ! grep -Fq 'mainnet-mining-authorization' params/readiness.go; then
  fail 'mainnet mining authorization gate must be encoded'
fi

if ! grep -Fq 'func MainnetReadiness()' params/readiness.go; then
  fail 'mainnet readiness gates must be encoded'
fi

if ! grep -Fq 'NetworkMainnet       NetworkID = "sudharma-mainnet-1"' params/network.go; then
  fail 'mainnet network identity must stay isolated'
fi

if grep -R --line-number 'MainnetLaunchAuthorized = true' params cmd blockchain 2>/dev/null; then
  fail 'mainnet launch must not be authorized in this freeze branch'
fi

grep -Fq 'func MiningReadiness()' params/mining_readiness.go \
  || fail 'mining readiness gates must be encoded in params/mining_readiness.go'

grep -Fq 'MiningStackReady' cmd/sudharma-mining-readiness/main.go \
  || fail 'sudharma-mining-readiness command must exist'

printf 'PASS: mainnet readiness contract is present\n'
