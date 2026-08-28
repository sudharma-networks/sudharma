#!/usr/bin/env bash
# Verify that CI retains the demand-miner safety gates.
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
workflow="$repo_root/.github/workflows/ci.yml"

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

require_contains() {
  local description=$1
  local needle=$2
  grep -Fq -- "$needle" "$workflow" || fail "CI is missing ${description}: ${needle}"
}

[[ -f "$workflow" ]] || fail "missing CI workflow: $workflow"

require_contains 'Go formatting check' 'gofmt -l'
require_contains 'focused demand-miner test' 'go test ./demandminer ./cmd/sudharma-demand-miner -count=1'
require_contains 'focused demand-miner race test' 'go test -race ./demandminer ./cmd/sudharma-demand-miner -count=1'
require_contains 'repository-wide race test' 'go test -race ./... -count=1'
require_contains 'Go vet check' 'go vet ./...'
require_contains 'demand-miner command build' 'go build -trimpath -o "$RUNNER_TEMP/sudharma-demand-miner" ./cmd/sudharma-demand-miner'
require_contains 'native miner command build' 'go build -trimpath -o "$RUNNER_TEMP/sudharmad" ./cmd/sudharmad'
require_contains 'installer safety test' 'bash deployment/testnet/install-demand-miner_test.sh'
require_contains 'systemd unit verification' 'systemd-analyze verify deployment/testnet/sudharma-demand-miner.service'
require_contains 'tracked secret scan' 'bash ./scripts/check-tracked-secrets.sh'
require_contains 'tracked secret scanner test' 'bash ./scripts/check-tracked-secrets_test.sh'

printf 'PASS: demand-miner CI safety gates are present\n'
