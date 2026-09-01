#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

go test ./pool -run TestStratumPoolRoundTripCreditsWorkerShare -count=1 >/dev/null \
  || fail 'pool stratum round-trip integration test must pass'

go test ./gpuminer/stratum/... ./pool/stratum/... -count=1 >/dev/null \
  || fail 'stratum client and server tests must pass'

printf 'PASS: pool mining smoke tests succeeded\n'
