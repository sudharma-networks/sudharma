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
require_literal 'preflight:'
require_literal 'deploy:'
require_literal 'aws-preflight:'
require_literal "if: github.event_name == 'workflow_dispatch' && (inputs.preflight == true || inputs.deploy == true)"
require_literal 'name: Verify AWS identity and faucet resources (read-only)'
require_literal 'aws sts get-caller-identity'
require_literal 'aws dynamodb describe-table'
require_literal 'aws secretsmanager describe-secret'
require_literal 'aws lambda get-function-configuration'
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

# Shared public RPC rollback safety. Capture the previously deployed Lambda ZIP
# before replacement and restore both code and environment if the pre-activation
# compatibility smoke test fails.
require_literal 'name: Snapshot current Lambda code for rollback'
require_literal 'aws lambda get-function --function-name "$LAMBDA_NAME"'
require_literal '/tmp/lambda-code-location.txt'
require_literal '/tmp/lambda-code-rollback.zip'
require_literal 'rollback_shared_lambda()'
require_literal '--zip-file fileb:///tmp/lambda-code-rollback.zip'
require_literal '--environment file:///tmp/lambda-environment-base-wrapper.json'
require_literal "trap 'rollback_shared_lambda' ERR"
require_literal 'name: Verify shared public RPC before faucet activation'

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

# OIDC/resource trust must be provable without changing AWS. Keep the dedicated
# preflight job read-only so a trust-policy diagnosis never becomes a rollout.
preflight_line="$(grep -n -m1 '^  aws-preflight:' "$workflow" | cut -d: -f1 || true)"
deploy_line="$(grep -n -m1 '^  deploy:' "$workflow" | cut -d: -f1 || true)"
[ -n "$preflight_line" ] || fail 'missing read-only aws-preflight job'
[ -n "$deploy_line" ] || fail 'missing deploy job'
if [ "$preflight_line" -ge "$deploy_line" ]; then
  fail 'aws-preflight job must appear before deploy job'
fi
preflight_block="$(sed -n "${preflight_line},$((deploy_line - 1))p" "$workflow")"
for forbidden in 'update-function-' 'put-' 'delete-' 'create-' 'send-command' 'start-' 'stop-' 'terminate-'; do
  if grep -Fq -- "$forbidden" <<<"$preflight_block"; then
    fail "aws-preflight must remain read-only; found forbidden mutation token: $forbidden"
  fi
done

stage_line="$(grep -n -m1 'name: Stage faucet disabled' "$workflow" | cut -d: -f1 || true)"
code_line="$(grep -n -m1 'name: Update Lambda code from tested artifact' "$workflow" | cut -d: -f1 || true)"
[ -n "$stage_line" ] || fail 'missing Stage faucet disabled step'
[ -n "$code_line" ] || fail 'missing Update Lambda code from tested artifact step'
if [ "$stage_line" -ge "$code_line" ]; then
  fail 'faucet must be forced disabled before new Lambda code is installed'
fi

# Stage 2 is a faucet recovery, not a general Lambda reconfiguration. Keep the
# function's existing runtime, handler, timeout and memory settings untouched.
stage_block="$(sed -n "${stage_line},$((code_line - 1))p" "$workflow")"
for forbidden in '--runtime ' '--handler ' '--timeout ' '--memory-size '; do
  if grep -Fq -- "$forbidden" <<<"$stage_block"; then
    fail "Stage faucet disabled must not rewrite unrelated Lambda setting: $forbidden"
  fi
done

printf 'PASS: faucet deployment contract is manual-only, has a read-only AWS/OIDC preflight, preserves shared Lambda routes/environment/configuration, rolls shared Lambda code/environment back on pre-activation smoke failure, is deep-health gated, fail-closed on unexpected errors, disables before code update, and promotes the tested artifact\n'
