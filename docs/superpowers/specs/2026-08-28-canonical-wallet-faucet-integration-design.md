# Canonical Wallet, Faucet, and Demand-Miner Integration Design

## Purpose

Establish `feature/public-testnet-wallet-v2` as the single reviewable integration line for the Sudharma Android wallet, permanent public-testnet RPC, faucet, 25-to-50 Test SUDH challenge, and demand-based testnet transaction confirmation. Preserve the existing draft branches and keep deployment, GPU consensus, economics, and Mainnet outside this integration checkpoint.

## Baseline

- Canonical base: `feature/public-testnet-wallet-v2` at `76ae28eaf8d5e9d84ba82302abd471b84304673a`.
- Main baseline: `main` at `5c925b47f04aacbbc2e239d0ce1f86357ffa3938`.
- Android-only history: `feature/android-wallet-v0.1`, open draft PR #20.
- Demand-miner follow-up history: `feature/demand-miner-v1`, open draft PR #21 targeting the canonical base.
- Combined review: open draft PR #23 targeting `main`.
- GPU-PoW history: `feature/gpu-pow-v1`, open draft PR #25; excluded from this design.

The canonical base already contains the Android wallet, public RPC Lambda, faucet and challenge implementation, a preliminary demand-miner integration, tests, deployment templates, and CI definitions. This project therefore consolidates and verifies existing work; it does not redesign those subsystems.

## Non-negotiable boundaries

- Mainnet remains disabled.
- Preserve the existing genesis, zero-premine policy, block subsidy, supply rules, transaction fee calculation, and original active consensus.
- Do not merge or activate GPU-PoW/Khushi consensus.
- Do not deploy to Seed-1 or Seed-2 during repository consolidation.
- Do not add public mining controls to the faucet or RPC proxy.
- Android signs user transactions locally; private keys and recovery phrases never leave the device.
- Faucet signing material remains outside Git and is retrieved only through narrowly scoped runtime secret access.
- Use short-lived GitHub OIDC for AWS; do not add long-lived AWS credentials or AdministratorAccess.
- Keep PRs #20, #21, #23, and #25 draft until their independent release gates pass.
- Do not delete or rewrite historical branches during consolidation.

## Canonical ownership

The combined integration line owns:

1. Android wallet behavior and protocol compatibility.
2. Permanent HTTPS public-testnet RPC endpoint policy.
3. One-time 100 Test SUDH faucet grant.
4. Challenge metadata and the exact 25 Test SUDH payment / 50 Test SUDH reward contract.
5. Replay protection, challenge-round limits, cooldown, and confirmed-transaction validation.
6. Demand-based mining code and disabled-by-default packaging needed to confirm pending public-testnet transactions.
7. CI assertions for all of the above.

PR #20 remains the historical Android review line. PR #21 remains the isolated demand-miner review line. PR #23 is the only combined integration review line. No change is copied from a historical branch merely because that branch is newer; each unique commit must be classified as already integrated, superseded, documentation-only, or a candidate supported by a failing test.

## Data flows

### Initial faucet grant

The Android wallet obtains live faucet metadata from the public HTTPS endpoint and requests the one-time grant for its locally derived address. The Lambda validates the address, reserves an idempotent grant record, obtains the dedicated faucet signer through its runtime boundary, creates and submits the signed payout, and records the result. The faucet source pays the protocol fee. The wallet polls authoritative transaction/account state and never reports confirmation solely from a successful HTTP request.

### Challenge

The wallet first requires the initial grant. It obtains the official challenge address and current constants from live metadata, locally signs a transaction sending exactly 25 Test SUDH, and submits it through RPC. After the transaction is confirmed, the wallet submits its transaction ID to the challenge endpoint. The Lambda verifies the confirmed transaction sender, recipient, and exact amount; reserves that transaction ID against replay; enforces cooldown and a maximum of five rounds; then submits exactly 50 Test SUDH back to the same wallet. A confirmed initial grant that was not fully reconciled in storage is reconciled before challenge validation.

Protocol transaction fees are paid according to the existing consensus calculation. The challenge deposit address is the faucet/development-controlled address in the present implementation; there is no separate invented application fee or consensus change in this project.

### Demand confirmation

The public faucet and wallet can only submit transactions. A separate private, single-instance service observes loopback RPC. It mines no blocks for an empty mempool and invokes one bounded native mining child when valid pending transactions exist. The service remains disabled after packaging and requires a later controlled single-host acceptance procedure before activation.

## Divergence reconciliation

Compare `feature/demand-miner-v1` against the canonical base by patch identity and file behavior, not commit count. The branch is currently both ahead of and behind the canonical base. Reconciliation must:

- preserve canonical faucet/challenge fixes;
- preserve later demand-miner hardening that is not already integrated;
- reject changes that overwrite shared node binaries, weaken runtime locking, follow unsafe configuration symlinks, mutate staging roots after refused activation, or broaden AWS permissions;
- retain the existing P2P test synchronization fix only when it changes test determinism and not production networking behavior;
- produce a linear, reviewable series of focused TDD commits on the canonical integration branch.

## Verification gates

Repository verification requires:

- formatting and full Go tests;
- `go vet ./...`;
- full Go race detector;
- demand-miner focused race tests and both command builds;
- installer safety tests;
- tracked-secret guard and production tracked-secret scan;
- local two-node rehearsal and public-testnet container smoke test;
- Lambda unit tests for routing, upstream failover, faucet, challenge, replay, reconciliation, uncertainty, and secret-safe errors;
- Android JVM unit tests, lint, protocol golden vectors, and debug APK build;
- exact GitHub Actions success at the candidate commit.

The current Work runtime lacks a Go toolchain, so Go verification must run in GitHub Actions unless a compatible local toolchain becomes available. This limitation must be reported, never hidden.

## Deployment gates

Repository completion does not authorize deployment. Later live acceptance requires:

1. Read-only health and version checks on both AWS seed nodes.
2. Read-only verification of the public RPC/faucet routes and narrowly scoped OIDC permissions.
3. A fresh exact APK installed on the OnePlus 11R.
4. Fresh-wallet auto-connect, initial grant, confirmation, 25-to-50 challenge, protocol fee observation, restart, offline, and recovery tests.
5. Demand miner installed disabled on exactly one selected seed host.
6. Controlled transaction confirmation, empty-mempool non-mining observation, and rollback proof.
7. Explicit evidence-based activation decision.

## Failure handling

- Uncertain faucet submission returns a retryable unavailable result and must not create an unguarded duplicate payout.
- Unconfirmed challenge transactions remain ineligible.
- A used challenge transaction ID cannot be claimed by another address or round.
- Malformed upstream or faucet responses fail closed with sanitized messages.
- If both public RPC upstreams fail, the wallet reports unavailable rather than stale success.
- Wrong network identity terminates demand mining.
- Installer preflight failure causes no partial activation or unsafe target mutation.
- Any failure in an exact candidate CI run blocks the release checkpoint even if an earlier commit passed.

## Completion definition

The repository consolidation stage is complete only when unique demand-miner divergence is classified, supported hardening is integrated through failing tests, the exact combined candidate passes all available verification, and PR #23 accurately documents what is implemented versus what remains undeployed. It is not a wallet release, AWS deployment, GPU activation, Kryptex milestone, or Mainnet authorization.
