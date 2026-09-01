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

require_literal 'ValidateShare' pool/share.go
require_literal 'SchemePPS' pool/payout.go
require_literal 'SchemePPLNS' pool/payout.go
require_literal 'SchemeSolo' pool/payout.go
require_literal 'SchemeFPPS' pool/payout.go
require_literal 'ParseWorkerIdentity' pool/worker.go
require_literal 'mining.subscribe' pool/stratum/server.go
require_literal 'mining.authorize' pool/stratum/server.go
require_literal 'mining.submit' pool/stratum/server.go
require_literal 'mining.notify' pool/stratum/server.go
require_literal 'pool.NewEngine' cmd/sudharma-pool/main.go
require_literal 'stratum.NewServer' cmd/sudharma-pool/main.go
require_literal 'payout_scheme' deployment/testnet/pool.example.json
require_literal 'stratum_listen' deployment/testnet/pool.example.json

require_literal 'ParsePoolURL' gpuminer/stratum/client.go
require_literal 'ParseNotify' gpuminer/stratum/job.go
require_literal 'LoadPoolFileConfig' gpuminer/poolfileconfig.go
require_file() {
  [ -f "$1" ] || fail "$1 is missing"
}

require_file 'deployment/testnet/gpu-miner-pool.example.json'
require_file 'deployment/mainnet/pool.example.json'
require_file 'deployment/mainnet/gpu-miner-pool.example.json'

go test ./pool -run TestStratumPoolRoundTripCreditsWorkerShare -count=1 >/dev/null \
  || fail 'pool stratum round-trip integration test must pass'

go test ./gpuminer/stratum/... -count=1 >/dev/null \
  || fail 'stratum miner client tests must pass'

go test ./pool/... ./cmd/sudharma-pool/... -count=1 >/dev/null \
  || fail 'pool mining tests must pass'

printf 'PASS: pool mining contract is present\n'
