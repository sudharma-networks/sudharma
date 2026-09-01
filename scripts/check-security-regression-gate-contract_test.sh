#!/usr/bin/env bash
set -euo pipefail

for file in \
  scripts/security-regression-gate.sh \
  scripts/generate-mainnet-genesis-candidate.sh \
  params/security_review_evidence.go \
  docs/audits/2026-09-01-public-security-review-window.md \
  docs/audits/2026-09-01-gpu-physical-evidence-checklist.md \
  docs/audits/2026-09-01-security-review-evidence-record.template.json; do
  [ -f "$file" ] || { echo "missing $file" >&2; exit 1; }
done

grep -Fq 'SecurityReviewEvidenceGates' params/security_review_evidence.go \
  || { echo 'SecurityReviewEvidenceGates missing' >&2; exit 1; }

grep -Fq 'security_review_gates' cmd/sudharma-mainnet-readiness/main.go \
  || { echo 'mainnet readiness must expose security_review_gates' >&2; exit 1; }

go test ./params -run 'TestSecurityReviewEvidenceSubGatesStayClosedByDefault|TestMainnetReadinessUsesEvidenceBasedSecurityReviewGate' -count=1 >/dev/null

printf 'PASS: security regression gate contract present\n'
