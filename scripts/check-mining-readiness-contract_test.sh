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
  params/mining_readiness.go \
  cmd/sudharma-mining-readiness/main.go \
  docs/audits/2026-08-31-pool-mining-architecture.md; do
  require_file "$path"
done

grep -Fq 'func MiningReadiness()' params/mining_readiness.go \
  || fail 'MiningReadiness must be encoded'

grep -Fq 'pool-stratum-stack' params/mining_readiness.go \
  || fail 'pool-stratum-stack gate must be encoded'

go test ./params -run TestMiningReadiness -count=1 >/dev/null \
  || fail 'mining readiness params tests must pass'

go test ./cmd/sudharma-mining-readiness -count=1 >/dev/null \
  || fail 'sudharma-mining-readiness tests must pass'

go run ./cmd/sudharma-mining-readiness >/dev/null \
  || fail 'sudharma-mining-readiness must report stack_ready'

printf 'PASS: mining readiness contract is present\n'
