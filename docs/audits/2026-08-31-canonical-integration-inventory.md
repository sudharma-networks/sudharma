# Canonical integration inventory

**Recorded:** 2026-08-31  
**Canonical base:** `main` at `633843d3719645cf7e81c51ff47cfaad5374c4c7`

This inventory is a source-control snapshot, not a deployment attestation.
The exact revisions running on the seeds, Lambda, demand miners, website, and
released APKs still require evidence from those systems before deployment or
release reconciliation can be declared complete.

## Safety gate

Canonical integration must not activate or deploy a component. The main CI
workflow runs `scripts/live-workflow-trigger-safety.test.mjs`, which requires
known AWS and public-testnet mutation workflows to be manual-only as soon as
they are introduced. Integration changes must preserve this check.

## Candidate source lines

| Area | Snapshot revision | Relationship to recorded `main` | Status |
| --- | --- | --- | --- |
| Faucet recovery, demand miner, and explorer deployment | `feature/faucet-recovery-stage2` at `2e2abd6786309fc1f4817e76b257eb632bd10fdc` | 159 commits ahead; recorded `main` is an ancestor | **Stage 1 complete on guard branch** — recovery workflows, contract scripts, shared deploy routes, Lambda recovery suite, and demand-miner funding/wake reconciled |
| Android wallet, faucet client, and earlier demand miner reconciliation | `codex/canonical-wallet-integration` at `6096172ef74775cf9aa68c32d0efba143400f61d` | 253 commits ahead and 82 behind | Integrated as the reviewed component spine; main's Telegram CI and manual-only mutation protections were retained |
| Explorer v1 | `feature/blockchain-explorer-v1` at `62f350e27bce545186df50fa72afed2c05c70282` | 72 commits ahead and 82 behind | **Stage 2 integrated via website-foundation** — Go explorer handlers, Lambda routes, contract script, manual-only deploy workflows |
| Website | `feature/website-foundation` at `ebc8f432c58aa7dbed381c68a0ae2b3ff5269747` | 90 commits ahead and 82 behind | **Stage 2 integrated on guard branch** — `web/` static site, visitor counter Lambda, website CI |
| GPU/Khushi staging | `feature/gpu-pow-v1` at `a3c5169f9eec28d7aae856bc31948f4606b6b900` | 373 commits ahead and 82 behind | Keep isolated and disabled by default; do not mix into the testnet service integration |
| Mainnet Tokenomics v1 | `feature/mainnet-tokenomics-v1` at `cdbe50e15ed5e81c238d15826a2534888b0c8c84` | 11 commits ahead; recorded `main` is an ancestor | Separate consensus review line; CI currently fails because `blockchain/rewards_test.go` references undefined `CreditMinerRewardFor` |

The wallet reconciliation and faucet recovery heads are not ancestors of one
another. Their common files therefore require a file-level comparison before
the recovery line's unique changes are integrated.

## Integration order

1. Preserve the manual-only workflow guard and secret checks.
2. Integrate the reviewed wallet, faucet, and demand-miner component spine.
3. Run Android, Go, race, Lambda, installer, workflow-contract, and
   secret-safety tests.
4. Reconcile the recovery line's newer faucet, demand-miner, RPC, website, and
   operational changes file by file, preserving manual-only workflows. **Stage 1
   is complete on the guard branch:** deep health, stale-prepared recovery,
   funding waits, demand-miner wake hooks, recovery operator workflows and
   contract scripts, shared deploy smoke routes (`/v1/website/visitors`,
   `/v1/explorer/status`), and demand-miner Go funding/wake modules are
   integrated. Full explorer API breadth and website foundation remain Stage 2.
5. Select the explorer API contract before integrating the website (Stage 2). **Complete on
   guard branch:** `website-foundation` chosen as contract source; Go seed handlers,
   Lambda proxy routes, `web/` site, visitor counter, and manual-only explorer deploy
   workflows integrated.
6. Verify the complete testnet candidate at one exact commit (Stage 3).
7. Keep GPU staging and Mainnet Tokenomics v1 on independent review lines.

## Stage 3 scope (next)

Verify the complete testnet candidate at one exact commit: attestation evidence,
release candidate tagging, and operator-gated deployment readiness review.

## Required evidence before testnet deployment

- Exact running commit or artifact digest for each seed service.
- Exact Lambda code/configuration revision and signer-independent health data.
- Exact demand-miner binary revision for both seeds.
- Exact website build revision.
- APK tag, source revision, and checksum for every published wallet.
- A reviewed diff from those deployed revisions to the canonical candidate.

No item in this inventory authorizes an AWS change, service restart, faucet
payout, mining action, release publication, or consensus activation.
