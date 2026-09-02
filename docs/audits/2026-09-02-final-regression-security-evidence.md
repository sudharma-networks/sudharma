# Sudharma final regression/security evidence — Stage 5 — 2026-09-02

## Classification

This is a **maintainer-controlled, AI-assisted internal evidence record** for Sudharma Network.

It is **not an independent third-party security audit, certification, or guarantee of security**.

## Reviewed main baseline

Exact merged `main` commit reviewed after Step 4:

```text
9882b46307b06fa78095103aab11d0f5a086d701
```

This commit includes the final owner-approved mainnet monetary-policy documentation from PR #116 while keeping mainnet launch/mining/genesis fail-closed.

## Final mainnet monetary-policy lock

The reviewed main branch consistently records the final mainnet policy as:

- exact maximum supply: **51,000,000 SUDH**
- premine: **0**
- target block interval: **60 seconds**
- subsidy-bearing blocks: **5,259,600**
- emission epochs: **40 quarterly epochs**, 131,490 blocks each
- nominal subsidy period: **10 target years**
- annual issuance shares: **16%, 14%, 13%, 12%, 11%, 10%, 8%, 7%, 5%, 4%**
- subsidy after height 5,259,600: **0 SUDH**

The existing public testnet remains on its separate legacy development monetary policy. The 51B testnet cap is not mainnet economics and is not rewritten by this evidence pass.

## Internal audit remediation mapping

| Audit item | Tracking issue | Final remediation | Status on reviewed main |
| --- | ---: | --- | --- |
| IS-001 fee overflow / fee conservation | PR #99 | merged audit remediation | Complete |
| IS-002 transaction atomicity | #100 | merged audit remediation | Complete |
| IS-003 tokenomics source-of-truth ambiguity | #101 | PR #116, merge `9882b46307b06fa78095103aab11d0f5a086d701` | Complete — final 51M mainnet policy |
| IS-004 cross-network transaction replay | #102 | PR #111, merge `5353948b8b244647e1c2eec3c80ee792d7548c41` | Complete — network-bound v2 signatures and two-way replay tests |
| IS-005 network-aware block/reorg/runtime processing | #103 | PR #109, merge `1033602d86ddcbbb194d85e4c15d6044155b818a` | Complete |
| IS-006 oversized P2P handshake `total_work` | PR #99 | bounded before big-int parse | Complete |
| IS-009 unbounded mempool/resource/dust economics | #104 | PR #114, merge `c9236ce3f2dfc397baafbea1c2d1526bd7e43841` | Complete |

Step 5 independently re-checked #102 after finding it had been reopened without a new technical finding. PR #111 and current-main network-aware verification/replay regression coverage were confirmed, and #102 was re-closed as completed on 2026-09-02.

## Exact post-Step-4 CI evidence

All three push workflows on reviewed main commit `9882b46307b06fa78095103aab11d0f5a086d701` completed successfully:

| Workflow | Run | Run ID | Result |
| --- | ---: | ---: | --- |
| CI | #1073 | `33594324731` | SUCCESS |
| Website CI | #173 | `33594324734` | SUCCESS |
| Faucet Recovery CI | #352 | `33594324750` | SUCCESS |

CI #1073 completed the full substantive engineering/security sequence successfully, including:

- tracked-secret safety checks
- Telegram bridge tests
- manual-only live mutation workflow enforcement
- public RPC Lambda tests
- mining/pool/readiness contract checks
- mainnet readiness contract
- mainnet go-live operator-toolkit contract
- mainnet merge-review contract
- mainnet monetary rehearsal
- pre-audit engineering selfcheck
- security-regression gate contract
- Go formatting
- focused demand-miner tests and race tests
- `go vet`
- demand-miner binary build and installer/systemd safety checks
- full Go test suite
- repository-wide race detector
- security regression/race/adversarial gate
- local two-node public-testnet rehearsal
- public-testnet container build
- public-testnet container smoke test

This supports the repository attestations that internal audit remediations and the automated security regression/race/adversarial gate have passed. It does **not** satisfy the physical-hardware or public-review sub-gates below.

## Fail-closed runtime state verified

On reviewed main:

```text
MainnetLaunchAuthorized = false
MainnetMiningAuthorized = false
MainnetGenesisTimestamp = 0
```

The default network remains `sudharma-testnet-1`; mainnet identity remains distinct as `sudharma-mainnet-1`.

## Remaining hard blockers

### 1. Physical GPU mining evidence — NOT COMPLETE

`PhysicalGPUMiningEvidenceComplete = false` remains correct.

Issue #24 records:

- RTX 2060 6 GiB checksum/canonical CUDA vector/benchmark/telemetry evidence: passed
- RTX 2060 independent localhost staging acceptance: **still required**
- physical AMD or other non-NVIDIA OpenCL GPU with at least 4 GiB dedicated VRAM: **still required**, including production memory/vector/benchmark and independent staging acceptance

Do not activate unrestricted mining or treat GPU evidence as complete until the retained issue #24 evidence satisfies those requirements.

### 2. Public/community security review — NOT COMPLETE

`PublicCommunitySecurityReviewComplete = false` remains correct.

The review window has not been frozen/started in this Stage 5 pass. A later exact candidate commit must be pinned when the public review is announced. The documented window recommends at least 14 days and must close with no unresolved Critical/High reports before this gate can be flipped.

### 3. Mainnet genesis/launch/mining authorization — NOT AUTHORIZED

Tokenomics finalization and internal remediation do not authorize launch. A later dedicated human-approved activation process must separately freeze the genesis timestamp/hash and only then consider launch/mining authorization after all readiness gates are complete.

## Stage 5 verdict

**Internal audit findings #101–#104: remediated and reconciled with current-main evidence.**

**Automated full-test/race/security/adversarial evidence: green on reviewed main.**

**Public testnet: CONTINUE WITH TESTNET CAUTION AND ABUSE MONITORING.**

**Mainnet: NO-GO.**

The remaining blockers are real evidence gates, not documentation placeholders: physical cross-vendor GPU/staging evidence, completed public/community security review, and the later explicit genesis/seed/launch authorization process. No later operator should flip those gates merely because this Stage 5 record exists.
