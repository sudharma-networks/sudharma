# Public/community security review window — 2026-09-01

## Purpose

Complete the zero-budget security-review evidence path without claiming an independent third-party audit.

## Window

| Field | Value |
| --- | --- |
| Candidate commit | **Not frozen yet.** Pin one exact post-remediation commit only when the review is publicly announced. |
| Review start (UTC) | **Not started.** Owner sets when announcing the pinned candidate. |
| Review end (UTC) | Owner sets at announcement; recommended minimum **14 days** after review start. |
| Reporting path | Repository `SECURITY.md` private vulnerability reporting |

The Step 5 evidence review on 2026-09-02 does **not** start this window. Do not set `PublicCommunitySecurityReviewComplete=true` merely because internal remediation and automated CI are green. If any consensus/security-critical change lands after a candidate is pinned, reviewers must decide whether the review window needs to restart against a new exact commit.

## Scope for reviewers

- internal audit report: `docs/audits/2026-09-01-internal-security-audit.md`
- Stage 5 evidence record: `docs/audits/2026-09-02-final-regression-security-evidence.md`
- network-bound signatures plan and merged PR #111
- network-aware consensus plan and merged PR #109
- mempool/resource bounds plan and merged PR #114
- final mainnet tokenomics source-of-truth: `docs/audits/2026-09-01-tokenomics-source-of-truth-kk.md` and merged PR #116
- physical Khushi GPU evidence status in issue #24

## Completion criteria

Flip `params.PublicCommunitySecurityReviewComplete` to `true` only when all are true:

1. One exact candidate commit and review-window dates are published.
2. The recommended minimum 14-day window has actually elapsed unless a longer published window was chosen.
3. No unresolved Critical/High reports remain at window close.
4. Accepted Medium/Low items are tracked with remediation or written risk acceptance.
5. Evidence record is stored from `docs/audits/2026-09-01-security-review-evidence-record.template.json`.
6. If the pinned commit changed materially during review, the final record explains why the existing window remains valid or records a restarted window.

## Accuracy requirement

Describe this process as **public/community security review**, not independent professional certification. A completed public/community review does not by itself certify that the software is vulnerability-free.
