# Sudharma Network Internal Security Audit — 2026-09-01

## Classification

This is a **maintainer-controlled, AI-assisted internal security audit** of Sudharma Network.

It is **not an independent third-party security audit** and must never be described or marketed as one.

Audit baseline:

```text
a02c67a85fd3c96f0183808504799889bc8f6dd4
```

Tracking branch: `audit/2026-09-01-internal-security`
Tracking PR: #99

## Executive verdict

**Public testnet: CONTINUE WITH NORMAL TESTNET CAUTION.**

**Mainnet: NO-GO.** Mainnet must remain fail-closed until the evidence gates in this report are complete.

The review found one High-severity monetary-arithmetic defect. A regression test first reproduced the failure, then the implementation was corrected and the repository CI/race/integration gates passed on the remediation branch. Two Medium findings remain tracked for hardening/documentation before the security-review evidence gate can close.

No claim is made that absence of additional findings proves absence of vulnerabilities.

## Scope reviewed

The internal audit reviewed or sampled the following security surfaces:

- transaction construction, signatures, nonces, replay protection and fees
- monetary-policy selection and supply-cap separation
- block validation and timestamp/future-time validation
- difficulty and chain-selection related code and existing tests
- staged state application and mempool validation
- wallet key generation, signature verification and encrypted wallet storage
- public-testnet/mainnet network separation and launch authorization guards
- GPU mining, pool/Stratum readiness gates and existing physical-test blockers
- faucet/RPC contracts covered by repository CI
- CI, race testing, two-node rehearsal and container smoke gates
- mainnet readiness/operator safeguards

## Method

1. Pin review to an exact canonical `main` commit.
2. Inspect consensus/security-critical source rather than relying on project documentation alone.
3. Convert reproducible findings into regression tests before implementation changes.
4. Require a failing test/check on the vulnerable implementation (RED).
5. Apply the minimal remediation and require repository CI to pass (GREEN).
6. Record unresolved findings as explicit mainnet blockers rather than silently accepting them.
7. Keep all mainnet authorization, genesis timestamp, seed topology and mining authorization gates closed.

## Findings

### IS-001 — HIGH — Transaction fee arithmetic overflow and non-conserving rounding

**Status: FIXED ON PR #99; awaiting merge.**

The baseline calculated basis-point fees by multiplying the full `uint64` amount before division. Amounts within the configured public-testnet/legacy monetary range could therefore overflow intermediate arithmetic.

The baseline also calculated total, development and miner fees as three independently floored percentages. For some ordinary atomic amounts the split did not equal the charged fee. For example, at 1000 atomic units the total 0.10% fee floors to 1 atom while the independently calculated 0.01% and 0.09% portions both floor to 0. `ApplyTransaction` subsequently requires the portions to equal the transaction fee, so this creates inconsistent validity/application behavior.

**Remediation:**

- Added security regression coverage at maximum configured legacy/testnet supply scale and small atomic values.
- Replaced full-width multiplication with quotient/remainder basis-point arithmetic.
- Defined miner fee as the exact charged-fee remainder after the development allocation so integer rounding always conserves fee atoms.

**Evidence:**

- Test-only commit: `8d181e6c5b9746c188574df1a48addf6738169a9` — expected CI failure reproduced the defect.
- First fixed commit line: `b8ffad367e7664a5f5df7982978477273bf52bbb` — CI passed.
- Final audit branch must remain green before merge.

### IS-002 — MEDIUM — `ApplyTransaction` discards credit failures / intrinsic atomicity is weak

**Status: OPEN — GitHub #100.**

`ApplyTransaction` checks the sender debit error but currently discards errors returned when crediting the receiver and development treasury. It also performs several mutations before nonce/processed markers are finalized.

Canonical mempool and block paths use cloned/temporary state before committing, which materially limits immediate exploitability. Nevertheless this helper should be intrinsically fail-closed so a future overflow/invariant failure cannot be silently accepted and a direct caller cannot be left with partial mutation.

Required before security-review closure:

- regression tests for receiver/development credit failure
- explicit atomicity contract
- aliasing tests (`From == To`, receiver/development overlap)
- fail-closed mutation/error propagation
- full race/regression verification

### IS-003 — MEDIUM — Testnet 51B vs mainnet 51M supply wording is ambiguous in top-level materials

**Status: OPEN — GitHub #101.**

The code intentionally separates monetary policies: development/public-testnet legacy policy uses a 51,000,000,000 SUDH hard cap while the current mainnet candidate policy encodes 51,000,000 SUDH. Some top-level wording can be read as though 51B applies universally, while mainnet-readiness material states 51M.

This review did not identify a confirmed code-path mix-up; it is a consensus-critical documentation/source-of-truth risk. The intended mainnet hard cap and emission schedule must be explicitly approved and consistently documented before genesis freeze.

### IS-004 — INFO/POSITIVE — Mainnet remains fail-closed

**Status: VERIFIED IN REVIEWED CODE/TESTS.**

Mainnet launch authorization remains false, the mainnet genesis timestamp remains unset, mainnet mining authorization remains false, and network identity is separated from the public testnet. The audit branch replaces the paid/independent-audit-only readiness item with a truthful `security-review-evidence` gate that also remains false.

This change does **not** authorize mainnet. The evidence gate requires the internal audit to have zero open Critical/High findings, required regression/adversarial tests to pass, and a documented public review period to complete.

### IS-005 — INFO/POSITIVE — Encrypted wallet storage uses authenticated encryption and memory-hard KDF parameters

**Status: NO CRITICAL DEFECT IDENTIFIED IN THIS PASS.**

The reviewed encrypted-wallet implementation uses random salt, scrypt, AES-256-GCM, a random nonce and authenticated envelope parameters. Unsupported envelope/KDF/cipher parameters are rejected. This finding is not a cryptographic certification and does not replace dedicated wallet penetration/recovery testing.

## CI / engineering evidence

The remediation branch is required to pass the repository security and engineering gates, including applicable:

- tracked-secret checks
- RPC/faucet contract checks
- mining/pool gates
- mainnet readiness tests
- `go vet`
- full Go test suite
- repository race detector
- two-node rehearsal
- public-testnet container build/smoke

The final PR head and workflow IDs should be recorded in the PR conversation before merge.

## Zero-budget security-review evidence gate

`MainnetSecurityReviewEvidenceComplete` must remain `false` until all of the following are satisfied:

- [ ] No open Critical findings.
- [ ] No open High findings.
- [ ] Every Medium finding is fixed or explicitly risk-accepted with written technical evidence.
- [ ] Full repository tests, race detector and required adversarial/security gates pass on the final candidate commit.
- [ ] Consensus/fork/reorg/difficulty/timestamp regression suite passes on the frozen candidate.
- [ ] Required RTX 2060 packaged localhost staging evidence is retained.
- [ ] Required physical AMD/non-NVIDIA OpenCL 4 GiB+ evidence is retained.
- [ ] Cross-vendor mining evidence receives independent/community reproducibility review where possible.
- [ ] A documented public security-review period is completed, with serious vulnerabilities routed privately through the repository security policy.
- [ ] Tokenomics/source-of-truth ambiguity is resolved before genesis freeze.
- [ ] Mainnet genesis timestamp/hash is frozen only after consensus-critical parameters are approved.
- [ ] Mainnet seed topology is deployed and verified separately from testnet.
- [ ] Mainnet launch/mining authorization is enabled only by an explicit final owner decision after every gate above is complete.

## Public/community review path

A paid auditor is not required for this internal evidence process. The project should invite developers and security researchers to review the frozen candidate and use private vulnerability reporting for high-impact findings. Community review should be described accurately as public/community review, not independent professional certification.

## Remaining project blockers outside this audit

Security-review completion is only one part of project finalization. Current known release blockers also include:

- Android wallet regression restoration/release work tracked in PR #98
- physical Khushi GPU evidence tracked in #24
- Kryptex-specific external onboarding/profile questions tracked in #13
- public documentation/community issues such as #41
- mainnet genesis/seed/launch gates described above

## Final security statement

Sudharma can continue development and public-testnet operation without purchasing an external audit. A future mainnet launch may use this evidence-based zero-budget route, provided the project remains transparent that no independent third-party audit occurred and does not close the security-review gate until its documented requirements are actually satisfied.
