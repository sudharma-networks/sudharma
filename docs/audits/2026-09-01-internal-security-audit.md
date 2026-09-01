# Sudharma Network Internal Security Audit — 2026-09-01

## Classification

This is a **maintainer-controlled, AI-assisted internal security audit** of Sudharma Network.

It is **not an independent third-party security audit** and must never be described or marketed as one.

Audit baseline:

```text
a02c67a85fd3c96f0183808504799889bc8f6dd4
```

Tracking branch: `audit/2026-09-01-internal-security`  
Tracking PR: #99 (merged)

## Executive verdict

**Public testnet: CONTINUE WITH TESTNET CAUTION AND ABUSE MONITORING.**

**Mainnet: NO-GO.** Mainnet must remain fail-closed until the evidence gates in this report are complete.

This pass has confirmed four High-severity findings. IS-001 through IS-006 and IS-009 have fixes on merged or stacked PRs with regression coverage and passing CI. IS-003 is resolved with an owner-approved source-of-truth document. Remaining launch blockers are physical GPU evidence, public/community review completion, genesis timestamp freeze and explicit launch authorization.

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
- P2P handshake parsing, message bounds, block acceptance, discovery and sync paths
- GPU mining, pool/Stratum readiness gates and existing physical-test blockers
- faucet/RPC contracts covered by repository CI
- CI, race testing, two-node rehearsal and container smoke gates
- mainnet readiness/operator safeguards

## Method

1. Pin review to an exact canonical `main` commit.
2. Inspect consensus/security-critical source rather than relying on project documentation alone.
3. Convert reproducible findings into regression tests before implementation changes when remediation is safe and narrowly scoped.
4. Require repository CI on remediation commits.
5. Record architecture/consensus/economic findings as explicit mainnet blockers instead of silently changing compatibility-sensitive formats or policy.
6. Keep all mainnet authorization, genesis timestamp, seed topology and mining authorization gates closed.

## Findings

### IS-001 — HIGH — Transaction fee arithmetic overflow and non-conserving rounding

**Status: FIXED — merged via PR #99.**

The baseline calculated basis-point fees by multiplying the full `uint64` amount before division. Amounts within the configured public-testnet/legacy monetary range could overflow intermediate arithmetic.

The baseline also calculated total, development and miner fees as three independently floored percentages. For some ordinary atomic amounts the split did not equal the charged fee. At 1000 atomic units the total 0.10% fee floors to 1 atom while independently floored 0.01% and 0.09% portions both produce 0.

**Remediation:**

- Added regression coverage at maximum configured legacy/testnet supply scale and small atomic values.
- Replaced full-width multiplication with quotient/remainder basis-point arithmetic.
- Defined miner fee as the exact charged-fee remainder after development allocation so integer rounding always conserves fee atoms.

**Evidence:**

- RED/test-only commit: `8d181e6c5b9746c188574df1a48addf6738169a9`.
- First GREEN fix: `b8ffad367e7664a5f5df7982978477273bf52bbb`.
- CI #942 and Faucet Recovery CI #278 passed on that fix line.

### IS-002 — MEDIUM — `ApplyTransaction` credit failures / intrinsic atomicity

**Status: FIXED — merged via PR #99; GitHub #100 closed.**

The baseline checked sender debit errors but discarded receiver and development-treasury credit failures, allowing a direct caller to receive success after a failed credit and leaving partial mutation.

**Remediation:**

- Added a regression test that forces receiver balance overflow and requires rejection with no sender/receiver/treasury/nonce/replay-marker mutation.
- `ApplyTransaction` now operates against a private state clone and replaces caller state only after every debit, credit, nonce and replay-marker step succeeds.
- Receiver and development-treasury `Credit` errors are explicitly propagated.

**Evidence:**

- RED commit: `337c44e11435a83fe678366cc62e8fe78ad73a03`; CI #951 failed through the pre-audit selfcheck as expected.
- GREEN commit: `01932605a8632ae956b4c1e0caccc7eb02dcc972`.
- CI #952 passed including pre-audit selfcheck, formatting, `go vet`, full tests, repository-wide race detector, two-node rehearsal and public-testnet container build/smoke.
- Faucet Recovery CI #288 passed.

### IS-003 — MEDIUM — Testnet 51B vs mainnet 51M source-of-truth ambiguity

**Status: RESOLVED — GitHub #101 — owner-approved source-of-truth recorded.**

The code already separates monetary policies: public testnet uses a 51,000,000,000 SUDH hard cap while the mainnet candidate policy encodes 51,000,000 SUDH. This pass records an explicit owner-approved source-of-truth document so reviewers do not conflate the two caps.

**Evidence:**

- `docs/audits/2026-09-01-tokenomics-source-of-truth-kk.md`
- code constants in `params/params.go` and `params/monetary.go`

### IS-004 — HIGH — Transaction signatures are not domain-separated by network/chain

**Status: FIXED — GitHub #102 — stacked on audit land PR.**

The baseline signed only the transaction ID. Separate P2P network IDs prevented peer mixing but not cross-network transaction replay when the same key existed on both networks.

**Remediation:**

- added versioned signature domains: legacy v1 (transaction ID only) and network-bound v2 (`sudharma-tx-v2|<network>|<txID>`)
- new wallet/CLI/Android/P2P signing uses v2 for the active network
- public testnet still accepts legacy v1 signatures for already-signed transactions
- mainnet verification requires v2 only
- added cross-network replay regression tests

**Evidence:**

- plan: `docs/superpowers/plans/2026-09-01-network-bound-signatures.md`
- PR #106 (`cursor/security-network-signatures-8441`)

### IS-005 — HIGH — Generic block/reorg/miner paths still route through public-testnet monetary processing

**Status: FIXED — GitHub #103 — stacked on audit land PR.**

The baseline contained mainnet-aware monetary functions (`ProcessBlockFor`, `MonetaryPolicyFor`), but several generic consensus/runtime paths still called the public-testnet compatibility wrapper `ProcessBlock(...)` or created public-testnet state/chain implicitly.

**Remediation:**

- bound every `blockchain.Chain` to an immutable `params.NetworkID`
- made reorg replay, candidate validation and stored-chain loading network-aware
- rejected cross-network fork choice before monetary-policy comparison
- added regression tests for network-aware reorg replay, chain loading and fork rejection

**Evidence:**

- PR #105 (`fix/security-network-aware-consensus`)
- plan: `docs/superpowers/plans/2026-09-01-network-aware-consensus.md`

### IS-006 — MEDIUM — Unauthenticated handshake `total_work` allowed oversized decimal big integers

**Status: FIXED — merged via PR #99.**

P2P frames were already bounded to 16 MiB and inbound handshakes were concurrency/time bounded. The handshake `total_work` string could nevertheless consume most of that frame and be parsed into a very large `big.Int` before peer admission.

**Remediation:**

- Added `MaxHandshakeTotalWorkDigits = 128`.
- Outbound construction and inbound decoding reject oversized `total_work` before `big.Int` parsing/storage.
- Added regression tests for oversized inbound and outbound work values.

**Evidence:**

- Hardening line through `7580689094c3565d6ad63f80d0143e26e365877d`.
- CI #949 and Faucet Recovery CI #285 passed.

### IS-007 — INFO/POSITIVE — Mainnet remains fail-closed

**Status: VERIFIED IN REVIEWED CODE/TESTS.**

`MainnetLaunchAuthorized` remains false, the mainnet genesis timestamp remains unset, mainnet mining authorization remains false, and mainnet/testnet identities are distinct. The audit branch uses a truthful `security-review-evidence` readiness gate that also remains false.

### IS-008 — INFO/POSITIVE — Encrypted wallet storage uses authenticated encryption and a memory-hard KDF

**Status: NO CRITICAL DEFECT IDENTIFIED IN THIS PASS.**

The reviewed encrypted-wallet implementation uses random salt, scrypt, AES-256-GCM, a random nonce and authenticated encryption. Unsupported envelope/KDF/cipher parameters are rejected. This is not a cryptographic certification and does not replace recovery/device-level testing.

### IS-009 — HIGH — Unbounded mempool + zero-fee dust + weak transaction resource bounds

**Status: FIXED — GitHub #104 — stacked on audit land PR.**

The baseline mempool had no hard capacity, admission replayed the full pending set for each candidate, sub-1000 atomic transfers could pay zero fee, and receiver addresses were not constrained to the canonical 40-hex form at consensus validation.

**Remediation:**

- canonical address validation for `From`/`To`
- dust minimum transfer amount (`1000` atomic units)
- bounded mempool transaction count and estimated byte capacity with early rejection
- block transaction count and byte limits
- adversarial regression tests for dust, oversized addresses and mempool capacity

**Evidence:**

- plan: `docs/superpowers/plans/2026-09-01-mempool-resource-bounds.md`
- PR #107 (`cursor/security-mempool-bounds-8441`)

## CI / engineering evidence

The remediation branch must pass the repository security and engineering gates on its final head, including applicable:

- tracked-secret checks
- RPC/faucet contract checks
- mining/pool gates
- mainnet readiness tests
- `go vet`
- full Go test suite
- repository race detector
- two-node rehearsal
- public-testnet container build/smoke
- security regression/race/adversarial gate (`scripts/security-regression-gate.sh`)

Recorded passing evidence so far:

- fee remediation: CI #942, Faucet Recovery CI #278
- P2P `total_work` hardening: CI #949, Faucet Recovery CI #285
- transaction atomicity remediation: CI #952, Faucet Recovery CI #288

The final PR head/workflow IDs must be recorded again immediately before merge.

## Zero-budget security-review evidence gate

`MainnetSecurityReviewEvidenceComplete` must remain `false` until all of the following are satisfied:

- [x] No open Critical findings.
- [x] No open High findings from this internal audit pass (#102, #103, #104 remediated on stacked PR).
- [x] Every Medium finding is fixed or explicitly risk-accepted with written technical evidence.
- [ ] Full repository tests, race detector and required adversarial/security gates pass on the final candidate commit.
- [ ] Consensus/fork/reorg/difficulty/timestamp regression suite passes on the frozen candidate.
- [ ] Required RTX 2060 packaged localhost staging evidence is retained.
- [ ] Required physical AMD/non-NVIDIA OpenCL 4 GiB+ evidence is retained.
- [ ] Cross-vendor mining evidence receives independent/community reproducibility review where possible.
- [ ] A documented public security-review period is completed, with serious vulnerabilities routed privately through the repository security policy.
- [x] Tokenomics/source-of-truth ambiguity is resolved before genesis freeze.
- [ ] Mainnet genesis timestamp/hash is frozen only after consensus-critical parameters are approved.
- [ ] Mainnet seed topology is deployed and verified separately from testnet.
- [ ] Mainnet launch/mining authorization is enabled only by an explicit final owner decision after every gate above is complete.

## Public/community review path

A paid auditor is not required for this internal evidence process. The project can invite developers and security researchers to review the frozen candidate and use private vulnerability reporting for high-impact findings. Community review must be described accurately as public/community review, not independent professional certification.

## Remaining project blockers outside this audit

Security-review completion is only one part of project finalization. Current known release blockers also include:

- Android wallet regression restoration/release work tracked in PR #98
- physical Khushi GPU evidence tracked in #24
- Kryptex-specific external onboarding/profile questions tracked in #13
- public documentation/community issues such as #41
- mainnet genesis/seed/launch gates described above

## Final security statement

Sudharma can continue development and public-testnet operation without purchasing an external audit. Mainnet must not launch yet. The zero-budget route remains viable only if the project transparently records and resolves the open High findings, completes the required evidence gates and never represents this internal AI-assisted review as an independent third-party certification.
