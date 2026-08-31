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
  params/monetary.go \
  params/mainnet_emission.go \
  docs/audits/2026-08-31-mainnet-readiness.md \
  docs/audits/2026-08-31-mainnet-launch-operator-runbook.md \
  cmd/sudharma-mainnet-readiness/main.go; do
  require_file "$path"
done

if ! grep -Fq 'MainnetLaunchAuthorized = false' params/network.go; then
  fail 'mainnet launch must stay unauthorized in params/network.go'
fi

if ! grep -Fq 'MainnetGenesisTimestamp uint64 = 0' params/network.go; then
  fail 'mainnet genesis timestamp must remain unset until human freeze'
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

printf 'PASS: mainnet readiness contract is present\n'
