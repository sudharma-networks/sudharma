#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

require_file() {
  [ -f "$1" ] || fail "$1 is missing"
}

for file in \
  docs/audits/2026-08-31-testnet-rc-attestation.md \
  docs/audits/2026-08-31-testnet-go-live-runbook.md \
  docs/audits/2026-08-31-testnet-go-live-operator-completion.md \
  deployment/testnet/deployment-evidence.template.json \
  scripts/generate-testnet-rc-attestation.sh \
  scripts/check-testnet-rc-readiness_test.sh \
  scripts/collect-testnet-deployment-evidence.mjs \
  scripts/verify-testnet-deployment-evidence.sh \
  scripts/verify-testnet-deployment-evidence.test.mjs; do
  require_file "$file"
done

for workflow in \
  .github/workflows/explorer-seed-rpc-deploy.yml \
  .github/workflows/explorer-public-rpc-deploy.yml \
  .github/workflows/testnet-public-rpc.yml \
  .github/workflows/demand-miner-auto-deploy.yml \
  .github/workflows/faucet-enable-public.yml \
  .github/workflows/provision-website-visitor-counter.yml; do
  require_file "$workflow"
  if grep -Eq '^[[:space:]]+(push|schedule|workflow_run|workflow_call):' "$workflow"; then
    fail "$workflow must remain manual-only for operator-gated go-live"
  fi
  grep -Fq 'workflow_dispatch:' "$workflow" || fail "$workflow must support workflow_dispatch"
done

node --test ./scripts/verify-testnet-deployment-evidence.test.mjs >/dev/null

printf 'PASS: testnet go-live operator toolkit is present and manual-only\n'
