#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

require_file() {
  [ -f "$1" ] || fail "$1 is missing"
}

for path in \
  docs/audits/2026-08-31-mainnet-merge-review-checklist.md \
  docs/audits/2026-08-31-pr76-reviewer-summary.md \
  docs/audits/2026-08-31-pr77-reviewer-summary.md \
  docs/audits/2026-08-31-owner-signoff-templates.md \
  docs/audits/2026-08-31-mainnet-genesis-freeze-template.md \
  docs/audits/2026-08-31-security-audit-evidence-template.md \
  docs/audits/2026-09-01-security-audit-kickoff-checklist.md \
  docs/audits/2026-09-01-security-audit-brief.md \
  docs/audits/2026-09-01-security-audit-scope.md \
  docs/audits/2026-09-01-security-audit-kickoff-record.template.json \
  deployment/mainnet/OPERATOR-CHECKLIST.md \
  deployment/testnet/pool-operator-runbook.md \
  deployment/testnet/windows-gpu-miner-republish-runbook.md \
  scripts/verify-mainnet-merge-readiness.sh \
  scripts/probe-testnet-mining-rpc.sh; do
  require_file "$path"
done

grep -Fq 'verify-mainnet-merge-readiness.sh' docs/audits/2026-08-31-mainnet-merge-review-checklist.md \
  || fail 'merge review checklist must reference unified verification script'

grep -Fq 'merge #77 before #76' docs/audits/2026-08-31-mainnet-merge-review-checklist.md \
  || fail 'merge order guard must be documented'

grep -Fq 'MainnetLaunchAuthorized = true' docs/audits/2026-08-31-mainnet-genesis-freeze-template.md \
  || fail 'genesis freeze template must document launch authorization gate'

grep -Fq 'confirm=PUBLISH' deployment/testnet/windows-gpu-miner-republish-runbook.md \
  || grep -Fq 'confirm`: `PUBLISH`' deployment/testnet/windows-gpu-miner-republish-runbook.md \
  || fail 'windows miner republish runbook must document workflow gate'

printf 'PASS: mainnet merge review contract is present\n'
