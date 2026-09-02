# Final Audit Evidence Stage 5 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reconcile the internal audit record with the fully merged Step 1–4 remediations and current-main CI evidence while keeping every genuinely incomplete mainnet security/activation gate fail-closed.

**Architecture:** This is an evidence/documentation-only finalization pass. It records current-main verification, corrects stale PR/status references, and clarifies that internal remediation and automated regression evidence are complete while physical GPU and public/community review evidence remain incomplete. No consensus constants, tokenomics, launch authorization, mining authorization, genesis timestamp, AWS, keys, seed topology, or live testnet state are changed.

**Tech Stack:** Markdown audit records, Go readiness-attestation comments/labels, GitHub Actions CI.

**Spec:** `docs/audits/2026-09-01-internal-security-audit.md` plus merged owner-approved tokenomics source-of-truth `docs/audits/2026-09-01-tokenomics-source-of-truth-kk.md`.

## Global Constraints

- Final mainnet hard cap remains exactly 51,000,000 SUDH.
- Mainnet emission remains 5,259,600 subsidy-bearing blocks / 40 quarterly epochs / nominal 10 target years.
- Public testnet legacy economics remain isolated and unchanged.
- `MainnetLaunchAuthorized` remains `false`.
- `MainnetMiningAuthorized` remains `false`.
- `MainnetGenesisTimestamp` remains `0`.
- `PhysicalGPUMiningEvidenceComplete` remains `false` until issue #24 evidence is genuinely complete.
- `PublicCommunitySecurityReviewComplete` remains `false` until a documented review window genuinely completes.
- The internal review must never be described as an independent third-party audit.

---

### Task 1: Record current-main final regression evidence

**Files:**
- Create: `docs/audits/2026-09-02-final-regression-security-evidence.md`

**Interfaces:**
- Consumes: merged main commit `9882b46307b06fa78095103aab11d0f5a086d701` and CI #1073 / Website #173 / Faucet Recovery #352.
- Produces: one canonical Step 5 evidence record for later genesis/readiness review.

- [ ] **Step 1:** Record exact commit, merged issue/PR mapping for #101–#104, and exact post-merge workflow IDs.
- [ ] **Step 2:** Record all successful substantive CI stages: monetary rehearsal, pre-audit selfcheck, gofmt, `go vet`, full tests, repository-wide race, security regression/race/adversarial gate, two-node rehearsal, container build/smoke.
- [ ] **Step 3:** Record fail-closed runtime flags and remaining hard blockers (#24 physical GPU evidence and public/community review).
- [ ] **Step 4:** State `Mainnet: NO-GO` and explicitly distinguish internal evidence from independent certification.

### Task 2: Reconcile the original internal audit report

**Files:**
- Modify: `docs/audits/2026-09-01-internal-security-audit.md`

**Interfaces:**
- Consumes: final evidence record from Task 1.
- Produces: accurate current finding statuses and evidence links.

- [ ] **Step 1:** Replace stale “stacked PR” references for IS-004/005/009 with actual merged PRs #111/#109/#114 and merge commits.
- [ ] **Step 2:** Update IS-003 from “candidate” wording to the final 51M / 10-target-year mainnet policy merged in PR #116.
- [ ] **Step 3:** Mark automated final-candidate/current-main CI/race/adversarial evidence complete based on CI #1073 while leaving physical/community/genesis/launch items incomplete.
- [ ] **Step 4:** Keep the executive verdict `Mainnet: NO-GO` and accurately list remaining blockers.

### Task 3: Correct security-review readiness evidence wording

**Files:**
- Modify: `params/security_review_evidence.go`
- Modify: `docs/audits/2026-09-01-public-security-review-window.md`

**Interfaces:**
- Consumes: current-main CI #1073 and open issue #24.
- Produces: truthful human-readable readiness details without changing boolean gate values.

- [ ] **Step 1:** Update comments/details for `InternalSecurityAuditRemediationComplete` and `SecurityRegressionRaceAdversarialGatePassed` to refer to merged/current-main evidence instead of an unfrozen candidate branch.
- [ ] **Step 2:** Do not change any of the four boolean values.
- [ ] **Step 3:** Replace the stale public-review candidate-commit placeholder with an explicit “not frozen / window not started” state and instructions to pin a later exact commit only when review begins.

### Task 4: Verify and merge the evidence-only pass

**Files:**
- Review all changed files from Tasks 1–3.

**Interfaces:**
- Consumes: Stage 5 branch head.
- Produces: merged evidence record with fresh CI proof.

- [ ] **Step 1:** Inspect the exact PR diff and confirm no consensus/runtime constants or authorization flags changed.
- [ ] **Step 2:** Require fresh branch CI success, including full tests/race/security gate and relevant Website/Faucet workflows.
- [ ] **Step 3:** Merge only with an exact-head SHA guard.
- [ ] **Step 4:** Verify post-merge `main` workflows and confirm remaining hard blockers are still false/open.
