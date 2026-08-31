#!/usr/bin/env bash
set -euo pipefail

workflow='.github/workflows/testnet-public-rpc.yml'

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[ -f "$workflow" ] || fail "$workflow is missing"

require_literal() {
  local needle="$1"
  grep -Fq -- "$needle" "$workflow" || fail "missing diagnostics-only deployment contract: $needle"
}

# A diagnostics-only rollout must be a distinct manual mode, not an alias for
# the normal faucet deployment that eventually enables payouts.
require_literal 'diagnostics_only:'
require_literal 'Deploy diagnostics while keeping the faucet disabled'
require_literal "if: github.event_name == 'workflow_dispatch' && (inputs.preflight == true || inputs.deploy == true || inputs.diagnostics_only == true)"
require_literal "if: github.event_name == 'workflow_dispatch' && (inputs.deploy == true || inputs.diagnostics_only == true)"
require_literal 'name: Validate deployment mode'
require_literal 'inputs.deploy == true && inputs.diagnostics_only == true'

# The existing activation path may remain available for a separately approved
# full rollout, but diagnostics-only mode must never execute it.
activation_line="$(grep -n -m1 'name: Activate faucet and smoke-test public readiness' "$workflow" | cut -d: -f1 || true)"
post_line="$(grep -n -m1 'name: Verify shared public RPC after faucet activation' "$workflow" | cut -d: -f1 || true)"
[ -n "$activation_line" ] || fail 'missing existing faucet activation step'
[ -n "$post_line" ] || fail 'missing existing post-activation verification step'
activation_block="$(sed -n "$((activation_line - 1)),$((post_line - 1))p" "$workflow")"
if ! grep -Fq -- "if: inputs.deploy == true && inputs.diagnostics_only != true" <<<"$activation_block"; then
  fail 'faucet activation must be skipped in diagnostics-only mode'
fi

verify_config_line="$(grep -n -m1 'name: Verify deployed Lambda configuration' "$workflow" | cut -d: -f1 || true)"
[ -n "$verify_config_line" ] || fail 'missing deployed Lambda configuration verification step'
post_block="$(sed -n "$((post_line - 1)),$((verify_config_line - 1))p" "$workflow")"
if ! grep -Fq -- "if: inputs.deploy == true && inputs.diagnostics_only != true" <<<"$post_block"; then
  fail 'post-activation verification must be skipped in diagnostics-only mode'
fi

# Diagnostics-only mode must prove that the tested artifact is live and that
# FAUCET_ENABLED is still false. Any failure in this verification must restore
# the previous Lambda code and environment from the existing rollback snapshot.
require_literal 'name: Verify diagnostics-only deployment remains fail-closed'
diag_line="$(grep -n -m1 'name: Verify diagnostics-only deployment remains fail-closed' "$workflow" | cut -d: -f1 || true)"
[ -n "$diag_line" ] || fail 'missing diagnostics-only verification step'
diag_block="$(sed -n "$diag_line,$((verify_config_line - 1))p" "$workflow")"
for required in \
  "if: inputs.diagnostics_only == true" \
  'rollback_shared_lambda()' \
  "trap 'rollback_shared_lambda' ERR" \
  '--zip-file fileb:///tmp/lambda-code-rollback.zip' \
  '--environment file:///tmp/lambda-environment-base-wrapper.json' \
  'expected_code_sha256=' \
  "--query 'Configuration.CodeSha256'" \
  "--query 'Environment.Variables.FAUCET_ENABLED'" \
  'FAUCET_ENABLED must remain false in diagnostics-only mode' \
  '$RPC_BASE_URL/v1/faucet/info' \
  'body.enabled !== false'; do
  if ! grep -Fq -- "$required" <<<"$diag_block"; then
    fail "diagnostics-only verification is missing: $required"
  fi
done

# The diagnostics-only verification must occur after the new code is installed
# and after the existing shared-route smoke test has succeeded.
code_line="$(grep -n -m1 'name: Update Lambda code from tested artifact' "$workflow" | cut -d: -f1 || true)"
shared_line="$(grep -n -m1 'name: Verify shared public RPC before faucet activation' "$workflow" | cut -d: -f1 || true)"
[ -n "$code_line" ] || fail 'missing Lambda code update step'
[ -n "$shared_line" ] || fail 'missing shared-route smoke step'
if [ "$diag_line" -le "$code_line" ] || [ "$diag_line" -le "$shared_line" ]; then
  fail 'diagnostics-only fail-closed verification must run after code update and shared-route smoke tests'
fi

printf 'PASS: diagnostics-only deployment is manual, mutually exclusive with full deployment, rollback-protected, verifies exact code identity, and keeps FAUCET_ENABLED=false\n'
