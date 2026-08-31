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
| Explorer v1 | `feature/blockchain-explorer-v1` at `62f350e27bce545186df50fa72afed2c05c70282` | 72 commits ahead and 82 behind | **Stage 2 complete** — integrated via website-foundation |
| Website | `feature/website-foundation` at `ebc8f432c58aa7dbed381c68a0ae2b3ff5269747` | 90 commits ahead and 82 behind | **Stage 2 complete** — `web/` site, visitor counter, website CI |
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
4. Reconcile the recovery line file by file (**Stage 1 complete**).
5. Integrate explorer API contract and website foundation (**Stage 2 complete**).
6. Verify the complete testnet candidate at one exact commit (**Stage 3 complete** on
   `5f92589`: RC readiness contract, attestation generator, explorer rehearsal smoke).
7. Operator-gated go-live toolkit (**Stage 4 complete** on guard branch): deployment
   evidence template, read-only public RPC collector, evidence verifier, and manual
   workflow runbook in `docs/audits/2026-08-31-testnet-go-live-runbook.md`.
8. Operator-gated Stage 5 go-live (**core complete**): seeds, public RPC, faucet enable,
   visitor counter provisioned; website static publish and Android APK release deferred.
9. Keep GPU staging and Mainnet Tokenomics v1 on independent review lines.

## Stage 5 status

Core Stage 5 go-live completed on 2026-08-31 (see
`docs/audits/2026-08-31-testnet-go-live-operator-completion.md`). Private evidence file
assembly is optional. Website Amplify publish and Android APK release remain operator
deferred.

## Stage 6 scope (current)

Public testnet surface hardening on the guard branch:

- Faucet web client and request UI against the live wallet proxy
- Browser CORS on faucet Lambda responses (deploy still operator-gated)
- Honest roadmap / home / testnet status copy for explorer + faucet
- Public API documentation for explorer and faucet surfaces

See `docs/audits/2026-08-31-stage6-public-surface-hardening.md`.

## Stage 7 scope (current)

Operator closure for the public surface:

- Live CORS verify script (`scripts/verify-faucet-browser-cors.mjs`)
- Manual `Promote Website Foundation` workflow for Amplify
- Port approved mainnet tokenomics website copy onto the guard branch

See `docs/audits/2026-08-31-stage7-public-surface-closure.md`.

## Required evidence before testnet deployment

- Exact running commit or artifact digest for each seed service.
- Exact Lambda code/configuration revision and signer-independent health data.
- Exact demand-miner binary revision for both seeds.
- Exact website build revision.
- APK tag, source revision, and checksum for every published wallet.
- A reviewed diff from those deployed revisions to the canonical candidate.

No item in this inventory authorizes an AWS change, service restart, faucet
payout, mining action, release publication, or consensus activation.
