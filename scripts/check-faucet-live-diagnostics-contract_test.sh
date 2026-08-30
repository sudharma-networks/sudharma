#!/usr/bin/env bash
set -euo pipefail

workflow='.github/workflows/faucet-live-diagnostics.yml'

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[ -f "$workflow" ] || fail "$workflow is missing"
[ -f scripts/sanitize-faucet-live-logs.mjs ] || fail 'missing live log sanitizer'

require_literal() {
  grep -Fq -- "$1" "$workflow" || fail "missing live diagnostics contract: $1"
}

require_literal 'uses: actions/checkout@v4'
require_literal 'node ./scripts/sanitize-faucet-live-logs.mjs'
require_literal "--query 'events[].message'"
require_literal 'DIAG_START_MS='
require_literal 'initial_prepared_at'
require_literal "date -u -d '6 hours ago'"

if grep -Fq -- '--output text; then' "$workflow"; then
  fail 'live diagnostics must not print unsanitized CloudWatch messages'
fi
if grep -Fq -- "date -u -d '2 hours ago'" "$workflow"; then
  fail 'live diagnostics must not use a sliding two-hour window that can miss the prepared payout'
fi

printf 'PASS: live diagnostics checkout the sanitizer and do not print raw CloudWatch messages\n'
