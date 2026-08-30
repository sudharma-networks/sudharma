#!/usr/bin/env bash
set -euo pipefail

workflow='.github/workflows/testnet-public-rpc.yml'
router='deployment/testnet/public-rpc/lambda/router.mjs'
upstream='deployment/testnet/public-rpc/lambda/upstream.mjs'

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[ -f "$workflow" ] || fail "$workflow is missing"
[ -f "$router" ] || fail "$router is missing"
[ -f "$upstream" ] || fail "$upstream is missing"

require_literal() {
  local needle="$1"
  local file="${2:-$workflow}"
  grep -Fq -- "$needle" "$file" || fail "missing required deployment contract in $file: $needle"
}

require_literal 'workflow_dispatch:'
require_literal 'deploy:'
require_literal "if: github.event_name == 'workflow_dispatch' && inputs.deploy == true"
require_literal 'uses: actions/download-artifact@v4'
require_literal "FAUCET_ENABLED: 'false'"
require_literal "FAUCET_ENABLED: 'true'"
require_literal '/v1/faucet/info'
require_literal '/v1/faucet/health'
require_literal "trap 'disable_faucet' ERR"
require_literal 'trap - ERR'
require_literal 'name: Snapshot Lambda environment'
require_literal '/tmp/lambda-environment-base.json'
require_literal '/tmp/lambda-environment-disabled.json'
require_literal '/tmp/lambda-environment-enabled.json'
require_literal '--environment file:///tmp/lambda-environment-disabled.json'
require_literal '--environment file:///tmp/lambda-environment-enabled.json'

# The public RPC Lambda is shared with website/explorer reads. A faucet recovery
# deployment must not regress routes already served by that Lambda.
require_literal 'visitor-runtime.mjs' "$workflow"
require_literal '/v1/website/visitors' "$router"
require_literal '/v1/explorer/status' "$router"
require_literal 'request.queryString' "$upstream"
require_literal 'curl -fsS "$RPC_BASE_URL/v1/website/visitors"' "$workflow"
require_literal 'curl -fsS "$RPC_BASE_URL/v1/explorer/status"' "$workflow"

if grep -Fq "if: github.ref == 'refs/heads/feature/public-testnet-wallet-v2'" "$workflow"; then
  fail 'deploy remains hard-wired to the historical feature/public-testnet-wallet-v2 branch'
fi

stage_line="$(grep -n -m1 'name: Stage faucet disabled' "$workflow" | cut -d: -f1 || true)"
code_line="$(grep -n -m1 'name: Update Lambda code from tested artifact' "$workflow" | cut -d: -f1 || true)"
[ -n "$stage_line" ] || fail 'missing Stage faucet disabled step'
[ -n "$code_line" ] || fail 'missing Update Lambda code from tested artifact step'
if [ "$stage_line" -ge "$code_line" ]; then
  fail 'faucet must be forced disabled before new Lambda code is installed'
fi

printf 'PASS: faucet deployment contract is manual-only, preserves and smoke-tests shared Lambda routes/environment, deep-health gated, fail-closed on unexpected errors, disables before code update, and promotes the tested artifact\n'
