#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

require_file() {
  [ -f "$1" ] || fail "$1 is missing"
}

require_file scripts/pre-audit-engineering-selfcheck.sh
[ -x scripts/pre-audit-engineering-selfcheck.sh ] || fail 'pre-audit selfcheck must be executable'

grep -Fq 'does NOT close audit gate' scripts/pre-audit-engineering-selfcheck.sh \
  || fail 'pre-audit selfcheck must state it does not close audit gate'

grep -Fq 'MainnetLaunchAuthorized stays false' scripts/pre-audit-engineering-selfcheck.sh \
  || fail 'pre-audit selfcheck must verify launch stays unauthorized'

require_file docs/audits/2026-09-01-pre-audit-engineering-selfcheck.md

bash ./scripts/pre-audit-engineering-selfcheck.sh >/tmp/sudharma-pre-audit-selfcheck.log \
  || fail 'pre-audit engineering selfcheck must pass on main'

grep -Fq 'pre_audit_selfcheck":"ok"' /tmp/sudharma-pre-audit-selfcheck.log \
  || fail 'pre-audit selfcheck must emit ok marker'

printf 'PASS: pre-audit engineering selfcheck contract\n'
