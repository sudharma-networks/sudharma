#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

require_literal() {
  local needle="$1"
  local file="$2"
  grep -Fq "$needle" "$file" || fail "$file must contain: $needle"
}

require_literal '/v1/mining/work' rpc/server.go
require_literal '/v1/mining/submit' rpc/server.go
require_literal 'handleMiningWork' rpc/gpu_mining.go
require_literal 'handleMiningSubmit' rpc/gpu_mining.go
require_literal 'buildPOWCompatWork' rpc/mining_compat.go
require_literal 'getblocktemplate' rpc/mining_compat.go
require_literal 'eth_getWork' rpc/mining_compat.go
require_literal 'miningWorkGet' deployment/testnet/public-rpc/lambda/router.mjs
require_literal 'miningSubmit' deployment/testnet/public-rpc/lambda/router.mjs
require_literal 'MainnetMiningAuthorized = false' params/mining.go
require_literal 'FailoverClient' gpuminer/failover_client.go

go test ./rpc -run 'TestBuildPOWCompatWork|TestGPUMinerWork' -count=1 >/dev/null \
  || fail 'GPU mining RPC tests must pass'

printf 'PASS: GPU mining API contract is present\n'
