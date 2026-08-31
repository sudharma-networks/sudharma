# Testnet release candidate attestation

**Recorded:** 2026-08-31  
**Candidate branch:** `cursor/canonical-integration-guard-8441`  
**Candidate commit:** `5f9258918fb301009a4e37ceb3f522906a8fd699` (regenerate with `bash scripts/generate-testnet-rc-attestation.sh` after later commits)

This document records source-control readiness for a public testnet release candidate.
It is **not** a deployment attestation. Live seed, Lambda, demand-miner, website, and APK
revisions still require operator evidence before any go-live action.

## Stage completion

| Stage | Scope | Status |
| --- | --- | --- |
| 0 | Canonical wallet/faucet/demand-miner spine | Complete |
| 1 | Faucet recovery reconciliation | Complete |
| 2 | Explorer API + website foundation | Complete |
| 3 | RC verification at one commit | Complete — CI green on `5f92589` |

## Contract verification (automated)

The RC generator runs these checks against the candidate tree:

- `check-canonical-faucet-recovery-contract_test.sh`
- `check-explorer-api-contract_test.sh`
- `check-faucet-deploy-contract_test.sh`
- `check-demand-miner-auto-deploy-contract_test.sh`
- `live-workflow-trigger-safety.test.mjs`

Main CI additionally runs Go tests, race detector, Lambda suite (62 tests), two-node
rehearsal with explorer smoke, and Android/website workflows on pull request.

## Rehearsal evidence

Local two-node rehearsal verifies:

- P2P peer retention and matching chain tip between seeds
- Seed explorer endpoints: `/v1/explorer/status`, `/v1/explorer/blocks`, `/v1/explorer/mempool`

Command:

```bash
bash ./scripts/testnet-rehearsal.sh
```

## Suggested release candidate tag

After merge review, operators may tag the verified commit:

```text
testnet-rc-2026-08-31
```

Tagging is optional until the candidate commit is frozen for deployment review.

## Deployment evidence still required before go-live

- Exact running commit or artifact digest for each seed service
- Exact Lambda code/configuration revision and signer-independent health data
- Exact demand-miner binary revision for both seeds
- Exact website build revision
- APK tag, source revision, and checksum for every published wallet
- Reviewed diff from deployed revisions to this candidate commit

## Operator authorization

Nothing in this attestation authorizes AWS changes, service restarts, faucet payouts,
mining actions, release publication, or consensus activation. All mutating workflows
remain `workflow_dispatch`-only.
