#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ci="$repo_root/.github/workflows/ci.yml"

require_literal() {
  local needle="$1"
  if ! grep -Fq -- "$needle" "$ci"; then
    echo "CI missing required demand-miner check: $needle" >&2
    exit 1
  fi
}

require_literal 'bash ./scripts/demand-miner-ci_test.sh'
require_literal 'bash ./deployment/testnet/install-demand-miner_test.sh'
require_literal 'bash ./scripts/check-tracked-secrets.sh'
require_literal 'go test -race ./demandminer ./cmd/sudharma-demand-miner -count=1'
require_literal 'go vet ./...'
require_literal 'go build -trimpath ./cmd/sudharma-demand-miner ./cmd/sudharmad'

echo "demand miner CI source checks passed"
