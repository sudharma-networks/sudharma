# Sudharma Network Internal Security Audit — 2026-09-01

## Classification

This is a **maintainer-controlled, AI-assisted internal security audit** of Sudharma Network.

It is **not an independent third-party security audit** and must never be described or marketed as one.

Original audit baseline:

```text
a02c67a85fd3c96f0183808504799889bc8f6dd4
```

Tracking branch: `audit/2026-09-01-internal-security`  
Tracking PR: #99 (merged)

## Stage 5 reconciliation — 2026-09-02

The original report is preserved as the audit record, but finding status/evidence below has been reconciled against the post-Step-4 `main` baseline:

```text
9882b46307b06fa78095103aab11d0f5a086d701
```

Canonical Stage 5 evidence record:

`docs/audits/2026-09-02-final-regression-security-evidence.md`

The Step 1–4 remediation sequence is now merged:

- #103 / IS-005 network-aware block/reorg/runtime processing — PR #109, merge `1033602d86ddcbbb194d85e4c15d6044155b818a`
- #102 / IS-004 network-bound transaction signatures — PR #111, merge `5353948b8b244647e1c2eec3c80ee792d7548c41`
- #104 / IS-009 bounded mempool/resource/dust policy — PR #114, merge `c9236ce3f2dfc397baafbea1c2d1526bd7e43841`
- #101 / IS-003 final 51M mainnet tokenomics reconciliation — PR #116, merge `9882b46307b06fa78095103aab11d0f5a086d701`

Issue #102 was found reopened during Stage 5 without a new technical finding or explanatory comment. The merged PR #111 implementation, current-main network-aware verification, two-way replay tests and current-main CI were re-verified; #102 was then re-closed as completed on 2026-09-02.

## Executive verdict

**Public testnet: CONTINUE WITH TESTNET CAUTION AND ABUSE MONITORING.**

**Mainnet: NO-GO.** Mainnet must remain fail-closed until every remaining evidence/activation gate is genuinely complete.

All required Critical/High/Medium remediations identified by this internal audit pass are now merged with regression evidence. The full current-main CI/race/security/adversarial sequence is green. This does **not** make mainnet ready: physical cross-vendor GPU/staging evidence and the public/community security-review window remain incomplete, and mainnet genesis/seed/launch/mining authorization remains separately human-gated.

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

1. Pin review to exact commits instead of relying on moving branch names.
2. Inspect consensus/security-critical source rather than relying on project documentation alone.
3. Convert reproducible findings into regression tests before implementation changes when remediation is safe and narrowly scoped.
4. Require repository CI on remediation heads and current merged main.
5. Record architecture/consensus/economic findings as explicit mainnet blockers instead of silently changing compatibility-sensitive formats or live-chain policy.
6. Keep all mainnet authorization, genesis timestamp, seed topology and mining authorization gates closed.

## Findings

### IS-001 — HIGH — Transaction fee arithmetic overflow and non-conserving rounding

**Status: FIXED — merged via PR #99.**

The baseline calculated basis-point fees by multiplying the full `uint64` amount before division. Amounts within the configured public-testnet/legacy monetary range could overflow intermediate arithmetic. The baseline also calculated total, development and miner fees as independently floored percentages, which could fail to conserve fee atoms.

**Remediation:**

- regression coverage at maximum configured legacy/testnet supply scale and small atomic values
- overflow-safe quotient/remainder basis-point arithmetic
- miner fee defined as the exact charged-fee remainder after development allocation

**Evidence:**

- RED/test-only commit: `8d181e6c5b9746c188574df1a48addf6738169a9`
- first GREEN fix: `b8ffad367e7664a5f5df7982978477273bf52bbb`
- CI #942 and Faucet Recovery CI #278 passed on that fix line

### IS-002 — MEDIUM — `ApplyTransaction` credit failures / intrinsic atomicity

**Status: FIXED — merged; GitHub #100 closed.**

The baseline checked sender debit errors but discarded receiver and development-treasury credit failures, allowing a direct caller to receive success after a failed credit and leaving partial mutation.

**Remediation:**

- regression test forcing receiver balance overflow with no state mutation allowed on rejection
- `ApplyTransaction` now mutates a private clone and commits only after all debit/credit/nonce/replay-marker steps succeed
- receiver and development-treasury `Credit` errors are propagated

**Evidence:**

- RED commit `337c44e11435a83fe678366cc62e8fe78ad73a03`; CI #951 failed as expected
- GREEN commit `01932605a8632ae956b4c1e0caccc7eb02dcc972`
- CI #952 and Faucet Recovery CI #288 passed

### IS-003 — MEDIUM — Testnet 51B vs mainnet 51M source-of-truth ambiguity

**Status: FIXED — GitHub #101 closed via PR #116.**

The running public testnet intentionally retains its legacy 51,000,000,000 SUDH development policy. The **final mainnet monetary policy** is separately locked at exactly **51,000,000 SUDH**, zero premine, 60-second target blocks, 5,259,600 subsidy-bearing blocks, 40 quarterly epochs and a nominal 10-target-year emission schedule. Subsidy is permanently zero after height 5,259,600.

**Remediation:**

- canonical source-of-truth explicitly labels 51B as legacy public-testnet-only
- README and website metadata label 51M as final mainnet economics
- documentation regression contract locks 51M / 40 epochs / 131,490 blocks per epoch / final height 5,259,600
- existing live public-testnet history/economics were not rewritten

**Evidence:**

- `docs/audits/2026-09-01-tokenomics-source-of-truth-kk.md`
- PR #116, merge `9882b46307b06fa78095103aab11d0f5a086d701`
- post-merge CI #1073, Website CI #173 and Faucet Recovery CI #352 all passed

### IS-004 — HIGH — Transaction signatures were not domain-separated by network/chain

**Status: FIXED — GitHub #102 closed; merged via PR #111.**

The baseline signed only the transaction ID. Separate P2P network IDs prevented peer mixing but did not prevent cross-network transaction replay when the same key/state existed on both networks.

**Remediation:**

- versioned signature domains: legacy v1 and network-bound v2 (`sudharma-tx-v2|<network>|<txID>`)
- new wallet/CLI/runtime/faucet signing uses the active network-bound domain
- public testnet accepts legacy v1 only for backward compatibility with already-signed transactions
- mainnet verification requires v2
- P2P/consensus validation verifies against the active network
- replay tests cover testnet -> mainnet and mainnet -> testnet
- Android regression coverage validates the network domain

**Evidence:**

- plan: `docs/superpowers/plans/2026-09-01-network-bound-signatures.md`
- PR #111, merge `5353948b8b244647e1c2eec3c80ee792d7548c41`
- PR-head CI #1003, Faucet Recovery CI #314 and Android Wallet CI #346 passed
- current-main CI #1073 re-ran the Go/security/race path successfully

### IS-005 — HIGH — Generic block/reorg/miner paths routed through public-testnet monetary processing

**Status: FIXED — GitHub #103 closed; merged via PR #109.**

The baseline contained mainnet-aware monetary functions but several generic consensus/runtime paths still used public-testnet compatibility wrappers or implicit public-testnet state/chain construction.

**Remediation:**

- chains/runtime paths bind to explicit network identity and monetary policy
- peer block acceptance, mining, stored-chain replay and state rebuild are network-aware
- reorg/candidate-chain validation uses the chain network/genesis policy
- cross-network fork choice is rejected
- regression coverage exercises testnet/mainnet network separation across runtime/replay/reorg paths

**Evidence:**

- plan: `docs/superpowers/plans/2026-09-01-network-aware-consensus.md`
- PR #109, merge `1033602d86ddcbbb194d85e4c15d6044155b818a`
- exact-head CI and post-merge Faucet Recovery evidence recorded in PR #109
- current-main CI #1073 passed full tests, race, two-node rehearsal and security gate

### IS-006 — MEDIUM — Unauthenticated handshake `total_work` allowed oversized decimal big integers

**Status: FIXED — merged via PR #99.**

P2P frames were already bounded, but the handshake `total_work` string could consume most of a frame and be parsed into a very large `big.Int` before peer admission.

**Remediation:**

- `MaxHandshakeTotalWorkDigits = 128`
- outbound construction and inbound decoding reject oversized `total_work` before `big.Int` parsing/storage
- regression tests cover oversized inbound and outbound values

**Evidence:**

- hardening line through `7580689094c3565d6ad63f80d0143e26e365877d`
- CI #949 and Faucet Recovery CI #285 passed

### IS-007 — INFO/POSITIVE — Mainnet remains fail-closed

**Status: VERIFIED ON STAGE 5 REVIEWED MAIN.**

```text
MainnetLaunchAuthorized = false
MainnetMiningAuthorized = false
MainnetGenesisTimestamp = 0
```

Public testnet remains the default network, and mainnet/testnet identities are distinct. `MainnetSecurityReviewEvidenceComplete()` remains false because physical GPU evidence and public/community review are still false.

### IS-008 — INFO/POSITIVE — Encrypted wallet storage uses authenticated encryption and a memory-hard KDF

**Status: NO CRITICAL DEFECT IDENTIFIED IN THIS PASS.**

The reviewed encrypted-wallet implementation uses random salt, scrypt, AES-256-GCM, a random nonce and authenticated encryption. Unsupported envelope/KDF/cipher parameters are rejected. This is not a cryptographic certification and does not replace recovery/device-level testing.

### IS-009 — HIGH — Unbounded mempool + zero-fee dust + weak transaction resource bounds

**Status: FIXED — GitHub #104 closed; merged via PR #114.**

The baseline mempool had no hard capacity, candidate admission replayed the full pending set, sub-1000 atomic transfers could pay zero fee, and receiver/resource bounds were weak.

**Remediation:**

- canonical address validation and transaction resource bounds
- explicit 1000-atomic minimum transfer
- bounded mempool transaction count/estimated bytes and per-sender cap
- sender/nonce index and sender-scoped pending validation on live admission paths
- block transaction count/byte limits
- adversarial tests for dust spam, oversized resources, duplicate nonce, capacity/accounting and explicit-network peer decode

**Evidence:**

- plan: `docs/superpowers/plans/2026-09-02-mempool-admission-bounds.md`
- PR #114, merge `c9236ce3f2dfc397baafbea1c2d1526bd7e43841`
- exact-head CI/Faucet Recovery evidence recorded on PR #114
- current-main CI #1073 passed full tests, repository-wide race and security regression/race/adversarial gate

## Current-main CI / engineering evidence

Reviewed main `9882b46307b06fa78095103aab11d0f5a086d701` passed:

- CI #1073 (`33594324731`)
- Website CI #173 (`33594324734`)
- Faucet Recovery CI #352 (`33594324750`)

CI #1073 successfully completed all substantive stages, including mainnet monetary rehearsal, pre-audit selfcheck, `gofmt`, `go vet`, full Go tests, repository-wide race detector, security regression/race/adversarial gate, local two-node testnet rehearsal, public-testnet container build and smoke test.

See `docs/audits/2026-09-02-final-regression-security-evidence.md` for the consolidated evidence record.

## Zero-budget security-review evidence gate

`MainnetSecurityReviewEvidenceComplete()` must remain `false` until every sub-gate and later activation prerequisite is genuinely satisfied:

- [x] No open Critical findings from this internal audit pass.
- [x] Required High findings #102, #103 and #104 are remediated/closed with merged regression coverage.
- [x] Medium findings are fixed; #101 now records the final 51M mainnet source-of-truth.
- [x] Full repository tests and repository-wide race detector pass on Stage 5 reviewed main.
- [x] Security regression/race/adversarial gate passes on Stage 5 reviewed main.
- [x] Mainnet monetary/readiness contracts pass on Stage 5 reviewed main.
- [ ] Re-run all required automated gates on the later exact frozen mainnet candidate if it differs from the reviewed commit.
- [ ] RTX 2060 independent packaged localhost staging acceptance is retained in #24.
- [ ] Physical AMD/non-NVIDIA OpenCL GPU with at least 4 GiB dedicated VRAM passes production memory/vector/benchmark and independent staging evidence in #24.
- [ ] Cross-vendor mining evidence receives reproducibility/community review where possible.
- [ ] A documented public/community security-review window is started against one exact candidate commit and completed with no unresolved Critical/High reports.
- [ ] Mainnet genesis timestamp/hash is frozen only after consensus-critical parameters and evidence gates are ready.
- [ ] Mainnet seed topology is prepared/verified separately from public testnet.
- [ ] Mainnet launch/mining authorization is enabled only by an explicit final owner decision after every required gate is complete.

## Public/community review path

A paid auditor is not required for this internal evidence process. The project can invite developers and security researchers to review a later pinned candidate and use private vulnerability reporting for high-impact findings. The review window is **not started by Stage 5**; see `docs/audits/2026-09-01-public-security-review-window.md`.

Community review must be described accurately as public/community review, not independent professional certification.

## Remaining project/mainnet blockers

The audit remediation sequence being complete is not equivalent to overall project or mainnet readiness. Current known blockers include:

- physical Khushi GPU evidence tracked in #24
- public/community security-review window not yet started/completed
- later exact frozen-candidate regression rerun
- mainnet genesis timestamp/hash freeze
- mainnet seed topology/operator verification
- explicit owner-controlled launch and mining authorization
- Android wallet regression restoration/release work tracked separately in PR #98 for the user-facing wallet baseline
- external pool/listing/onboarding work such as #13 where applicable

## Final security statement

Sudharma can continue development and public-testnet operation without purchasing an external audit. The maintainer-controlled internal audit findings identified in this pass have been remediated with current-main automated evidence, but **mainnet must not launch yet**. The zero-budget route remains viable only if the remaining physical GPU, public/community review, frozen-candidate, genesis/seed and explicit activation gates are completed truthfully. This internal AI-assisted review must never be represented as independent third-party certification.
