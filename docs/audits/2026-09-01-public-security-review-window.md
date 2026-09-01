# Public/community security review window — 2026-09-01

## Purpose

Complete the zero-budget security-review evidence path without claiming an independent third-party audit.

## Window

| Field | Value |
| --- | --- |
| Candidate commit | Record merge commit of `cursor/land-audit-stack-8441` |
| Review start (UTC) | _Owner sets when announcing_ |
| Review end (UTC) | _Owner sets (recommended minimum 14 days)_ |
| Reporting path | Repository `SECURITY.md` private vulnerability reporting |

## Scope for reviewers

- internal audit report: `docs/audits/2026-09-01-internal-security-audit.md`
- network-bound signatures plan
- mempool resource bounds plan
- tokenomics source-of-truth: `docs/audits/2026-09-01-tokenomics-source-of-truth-kk.md`

## Completion criteria

Flip `params.PublicCommunitySecurityReviewComplete` to `true` only when all are true:

1. Review window dates are published
2. No unresolved Critical/High reports remain at window close
3. Accepted Medium/Low items are tracked with remediation or written risk acceptance
4. Evidence record stored from `docs/audits/2026-09-01-security-review-evidence-record.template.json`

## Accuracy requirement

Describe this process as **public/community security review**, not independent professional certification.
