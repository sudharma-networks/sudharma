#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

require_literal() {
  grep -Fq -- "$1" "$2" || fail "missing in $2: $1"
}

for file in \
  blockchain/explorer.go \
  blockchain/explorer_test.go \
  rpc/explorer.go \
  rpc/explorer_cursor.go \
  rpc/explorer_test.go \
  rpc/explorer_pagination_test.go \
  docs/superpowers/specs/2026-08-29-blockchain-explorer-v1-design.md \
  deployment/testnet/public-rpc/lambda/shared-routes-regression.test.mjs \
  web/lib/explorer-api.ts \
  web/lib/explorer-config.ts; do
  [ -f "$file" ] || fail "$file is missing"
done

require_literal '/v1/explorer/status' rpc/server.go
require_literal '/v1/explorer/mempool' rpc/server.go
require_literal '/v1/explorer/blocks' rpc/server.go
require_literal '/v1/explorer/transactions' rpc/server.go
require_literal '/v1/explorer/search' rpc/server.go
require_literal '/v1/explorer/addresses/' rpc/server.go
require_literal 'handleExplorerMempool' rpc/explorer.go
require_literal 'explorerMempool' deployment/testnet/public-rpc/lambda/router.mjs
require_literal 'websiteVisitorsRead' deployment/testnet/public-rpc/lambda/router.mjs

for workflow in \
  .github/workflows/explorer-public-rpc-deploy.yml \
  .github/workflows/explorer-seed-rpc-deploy.yml; do
  [ -f "$workflow" ] || fail "$workflow is missing"
  require_literal 'workflow_dispatch:' "$workflow"
  if grep -Eq '^[[:space:]]+(push|schedule|workflow_run|workflow_call):' "$workflow"; then
    fail "$workflow must be manual-only"
  fi
done

printf 'PASS: explorer API contract files and manual-only deploy workflows are present\n'
