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
require_literal 'aws lambda get-function'
require_literal '--function-name "$LAMBDA_NAME"'
require_literal '/tmp/lambda-code-location.txt'
require_literal '/tmp/lambda-code-rollback.zip'
require_literal 'rollback_shared_lambda()'
require_literal '--zip-file fileb:///tmp/lambda-code-rollback.zip'
require_literal '--environment file:///tmp/lambda-environment-base-wrapper.json'
require_literal "trap 'rollback_shared_lambda' ERR"
require_literal 'name: Verify shared public RPC before faucet activation'

# A late shared-route regression after the faucet becomes ready must also fail
# closed. The post-activation verification step must disable the faucet and
# restore the prior Lambda code/environment using the same rollback snapshot.
require_literal 'name: Verify shared public RPC after faucet activation'
post_line="$(grep -n -m1 'name: Verify shared public RPC after faucet activation' "$workflow" | cut -d: -f1 || true)"
verify_config_line="$(grep -n -m1 'name: Verify deployed Lambda configuration' "$workflow" | cut -d: -f1 || true)"
[ -n "$post_line" ] || fail 'missing post-activation shared RPC verification step'
[ -n "$verify_config_line" ] || fail 'missing deployed Lambda configuration verification step'
post_block="$(sed -n "${post_line},$((verify_config_line - 1))p" "$workflow")"
if ! grep -Fq -- 'rollback_shared_lambda()' <<<"$post_block"; then
  fail 'post-activation shared RPC verification must define rollback_shared_lambda'
fi
if ! grep -Fq -- "trap 'rollback_shared_lambda' ERR" <<<"$post_block"; then
  fail 'post-activation shared RPC verification must trap ERR and rollback'
fi
if ! grep -Fq -- '--environment file:///tmp/lambda-environment-base-wrapper.json' <<<"$post_block"; then
  fail 'post-activation rollback must restore original Lambda environment'
fi
if ! grep -Fq -- '--zip-file fileb:///tmp/lambda-code-rollback.zip' <<<"$post_block"; then
  fail 'post-activation rollback must restore previous Lambda code'
fi
if ! grep -Fq -- '/tmp/status-post.json' <<<"$post_block" || ! grep -Fq -- "s.network !== 'sudharma'" <<<"$post_block"; then
  fail 'post-activation status smoke must validate Sudharma network payload, not only HTTP success'
fi
if ! grep -Fq -- '/tmp/visitors-post.json' <<<"$post_block" || ! grep -Fq -- 'Number.isSafeInteger(v.total)' <<<"$post_block"; then
  fail 'post-activation visitor smoke must validate the visitor payload, not only HTTP success'
fi
if ! grep -Fq -- '/tmp/explorer-status-post.json' <<<"$post_block" || ! grep -Fq -- "typeof e !== 'object'" <<<"$post_block"; then
  fail 'post-activation explorer smoke must validate the explorer payload, not only HTTP success'
fi

# Promotion is not complete until AWS reports the same CodeSha256 as the exact
# ZIP downloaded from this workflow run. Keep this inside the rollback trap so
# a code-identity mismatch restores the previous shared Lambda.
if ! grep -Fq -- 'expected_code_sha256=' <<<"$post_block"; then
  fail 'post-activation verification must compute the expected SHA-256 of the tested Lambda ZIP'
fi
if ! grep -Fq -- "--query 'Configuration.CodeSha256'" <<<"$post_block"; then
  fail 'post-activation verification must read the deployed Lambda CodeSha256'
fi
if ! grep -Fq -- 'deployed_code_sha256=' <<<"$post_block"; then
  fail 'post-activation verification must capture the deployed Lambda CodeSha256'
fi
if ! grep -Fq -- 'deployed_code_sha256" != "$expected_code_sha256' <<<"$post_block"; then
  fail 'post-activation verification must compare deployed Lambda code identity with the tested artifact'
fi

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
if ! grep -Fq -- 'aws lambda get-function' <<<"$preflight_block"; then
  fail 'aws-preflight must prove lambda:GetFunction permission required for rollback snapshot'
fi
if ! grep -Fq -- 'Configuration.CodeSha256' <<<"$preflight_block"; then
  fail 'aws-preflight get-function must query safe metadata instead of exposing Code.Location'
fi
if grep -Fq -- 'Code.Location' <<<"$preflight_block"; then
  fail 'aws-preflight must not expose the presigned Lambda Code.Location URL'
fi

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

printf 'PASS: faucet deployment contract is manual-only, has a read-only AWS/OIDC preflight that proves rollback permissions, preserves shared Lambda routes/environment/configuration, validates shared-route payloads, verifies deployed code identity, rolls shared Lambda code/environment back on pre- and post-activation smoke failure, is deep-health gated, fail-closed on unexpected errors, disables before code update, and promotes the tested artifact\n'
