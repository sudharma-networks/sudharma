# Khushi Reference Verifier Reconciliation Design

**Date:** 2026-09-03  
**Tracking:** #118 Stage 2  
**Base:** `c523ff7a407b8ad442f15d175dfddb6bf9086b00`  
**Status:** Approved continuation of the staged pre-mainnet readiness path

## Goal

Port the reviewed deterministic Khushi / `sudharma-gpupow-v1` CPU reference verifier and canonical vectors from the diverged draft PR #25 onto current `main`, then connect that verifier to the Stage-1 `blockchain.ProofVerifier` interface without activating Version-2 proof of work on public testnet or mainnet.

This stage makes the current node code capable of independently verifying Khushi Version-2 proofs under an explicitly supplied finite `blockchain.PoWPolicy`. It does not cause the default network constructors to accept Version-2 blocks.

## Safety boundary

The following invariants remain unchanged throughout Stage 2:

- `params.MainnetLaunchAuthorized = false`
- `params.MainnetMiningAuthorized = false`
- `params.MainnetGenesisTimestamp = 0`
- `params.DefaultNetwork = params.NetworkPublicTestnet`
- `params.GPUV1TestnetActivationHeight = params.GPUV1ActivationDisabled`
- `params.GPUV1MainnetActivationHeight = params.GPUV1ActivationDisabled`
- `params.PhysicalGPUMiningEvidenceComplete = false`
- `params.PublicCommunitySecurityReviewComplete = false`
- no Seed-1 / Seed-2 mutation
- no AWS, IAM, key, faucet, wallet or public-RPC deployment mutation
- no public Stratum listener exposure
- no CPU production-mining path

Stage 2 is consensus-readiness engineering only. It is not an independent third-party security audit or certification.

## Compatibility requirements

1. Accepted Version-1 public-testnet blocks remain byte-for-byte and proof-for-proof compatible.
2. `pow.HashBlock` and `pow.CheckBlock` remain the legacy double-SHA256 Version-1 path.
3. Default public-testnet/mainnet `PoWPolicyForNetwork` values remain activation-disabled.
4. Default chain construction continues to use the Stage-1 legacy verifier and therefore cannot accept Version-2 while activation is disabled.
5. A Version-2 block is only testable through an explicit finite policy plus a Khushi-capable verifier.
6. Mainnet economics remain the already frozen 51,000,000 SUDH policy; Stage 2 does not touch issuance, subsidy, fees, addresses, genesis or transaction rules.

## Source provenance

The algorithm primitives and deterministic fixtures are reconciled from draft PR #25 only as reviewed reference material. The old branch is 373 commits / 250 files and is not mergeable wholesale.

Port only the `pow` CPU-reference algorithm and deterministic vectors needed for consensus verification. Do not port old activation, RPC, miner, CUDA/OpenCL, seed, deployment, Stratum or monetary-policy changes in this stage.

Any old documentation containing obsolete 100M-era tokenomics is non-authoritative and must not be copied into current-main policy.

## Architecture

### 1. Consensus-local Khushi contract

The consensus verifier must not depend on `compatibility/gpupowv1` or miner-packaging code. Necessary verification constants live under `pow`.

Freeze these reviewed Khushi v1 contract values:

- algorithm ID: `sudharma-gpupow-v1`
- epoch length: 7,500 blocks
- program period: 3 blocks
- lanes: 16
- registers per lane: 32
- cache accesses per round: 11
- math operations per round: 18
- DAG rounds: 64
- dataset parents per generated item: 512
- cache node size: 64 bytes
- production verifier cache bytes: 16 MiB
- production verifier cache nodes: 262,144

The 2 GiB GPU dataset, 256 MiB runtime reserve/chunk policy and 4 GiB physical-VRAM eligibility remain hardware/miner concerns unless a deterministic cross-implementation test proves a particular value is consensus-visible. Stage 2 must not silently promote hardware-only values into consensus.

### 2. Deterministic reference primitives

Reconcile the reviewed pure Go primitives:

- epoch derivation and domain-separated epoch seed
- Keccak-512 epoch cache generation
- deterministic dataset-item derivation
- KISS99 scheduling and FNV/FNV1a primitives
- register permutations
- lane initialization
- random math and merge families
- programmatic lane/group evaluation
- lane reduction
- final domain-separated digest
- canonical block-header-prefix + little-endian nonce binding

The existing canonical block serialization remains authoritative. No new block field is introduced.

### 3. Reference API

Keep legacy functions unchanged and add explicit Version-2 helpers:

- `GPUV1EpochForHeight(height)`
- `GPUV1EpochSeed(epoch)`
- `GPUV1BuildCache(seed, nodeCount)`
- `GPUV1DatasetItem(cache, index)`
- `GPUV1ReferenceDigest(headerPrefix, nonce, height, cache)` or equivalent exported test/reference boundary
- `GPUV1HashBlockWithCache(block, nonce, cache)`
- `GPUV1CheckBlockWithCache(block, cache)`

Export only what is needed for stable reference-vector verification or later independent implementations. Internal scheduling helpers remain unexported.

### 4. Chain proof verifier

Add a `chainProofVerifier` in `pow` implementing `blockchain.ProofVerifier`.

`NewChainProofVerifier(policy)`:

- validates the policy/verifier configuration
- supports Version 1 and Version 2
- does not mine
- lazily builds at most one production epoch cache at a time
- protects cache replacement with a mutex
- returns an error rather than falling back to a weaker proof path

Verification dispatch:

- Version 1 -> existing `CheckBlock`
- Version 2 -> Khushi reference verification using the cache for the block height
- all other versions -> reject
- any version disallowed by the supplied immutable `PoWPolicy` -> reject before expensive Khushi work

### 5. Cache lifecycle and denial-of-service boundary

Generating a 16 MiB cache is deterministic but computationally non-trivial. The verifier therefore keeps one current epoch cache and rebuilds only when the block height crosses an epoch boundary.

Tests use compact fixed caches (for example 8 nodes) to validate primitive vectors quickly. Tests that exercise `NewChainProofVerifier` must avoid repeatedly rebuilding the 262,144-node production cache. Provide a package-private constructor that accepts a test cache-node count so behavior can be proven with bounded fixtures while production construction stays fixed.

Concurrent verification must be race-safe. Cache generation must not mutate a cache returned to verification after publication.

### 6. Canonical vectors

Reconcile the reviewed compact interoperability vectors, retaining the statement that `cache_nodes=8` is a test fixture and not the production cache size.

Canonical compact digests include:

- `genesis-program-zero`: `2a7c15fc6c84a67d43ff7074ac5835aa433145f89d10d1d9e36a99fe22da4b2b`
- `program-zero-max-nonce`: `c936cf90a07dc7e63a8245729c2d1739ab51d18c8c6ada5bbf27aa13f65fa5ee`
- `program-one-boundary`: `a07fa24d81f154a7ec4381d0482473c164eb8edaafc868cdeb9c493e90a00d5e`
- `epoch-zero-tail`: `53edb109ee5b992123466a1f094f489962b7c5c9bfddd92a6741bf18718db062`
- `epoch-one-head`: `05767a914a67b5b5163c25bf73204bfad62afe7bb22feacd57b79449e97c7e74`
- `locked-reference`: `b5d8c0c458d37eb551e8e845488fec756ca02c8eba891bc104c4f9c426c96170`

Canonical Version-2 block-vector digest:

`cd5b04ebfebaa01656e34d9a34fb8b2a979d149287515119782477f103e78641`

The fixture metadata will use the actual consensus-visible identifier `sudharma-gpupow-v1`; explanatory text may call it the CPU reference implementation, but no alternate algorithm identity is introduced.

### 7. Negative and replay coverage

Stage 2 must prove rejection for:

- nil blocks
- Version 2 under legacy-only policy
- Version 1 after an explicit finite activation boundary
- Version 2 before the explicit finite activation boundary
- unknown block versions
- changed nonce
- changed header field / miner address / previous hash / merkle root / height / timestamp / difficulty
- wrong epoch cache
- empty cache
- digest above target
- verifier policy mismatch

Also prove that a valid Version-1 block remains accepted before activation by the new verifier.

### 8. Explicit finite-policy integration test

Build an isolated in-memory chain with `NewChainForWithConsensus` and an explicit finite test policy. This is the only Stage-2 path that exercises Version-2 chain validation.

Do not change network activation constants. Do not wire the new verifier into default node startup yet; that wiring belongs in the next integration stage after the reference verifier is independently green.

## Testing strategy

Use TDD for each layer:

1. contract/epoch tests RED -> primitive constants and seed GREEN
2. cache/dataset vectors RED -> cache/data derivation GREEN
3. program/mix/final vectors RED -> reference evaluator GREEN
4. JSON/canonical block vectors RED -> block reference verification GREEN
5. chain verifier policy/cache tests RED -> `NewChainProofVerifier` GREEN
6. finite-policy blockchain integration and negative/replay tests RED -> GREEN
7. `go test ./pow ./blockchain -count=1`
8. `go test -race ./pow ./blockchain -count=1`
9. repository full CI, race detector, security/adversarial gate and two-node/container rehearsal

## Merge gate

Stage 2 is mergeable only when:

- all canonical compact vectors match exactly
- canonical Version-2 block vector matches exactly
- Version-1 compatibility tests pass
- finite-policy Version-2 acceptance/rejection tests pass
- default activation remains disabled
- no `compatibility` dependency is introduced into consensus verification
- full exact-head push CI is green
- exact-head PR CI and Faucet Recovery CI are green
- diff review confirms no live infrastructure or monetary-policy changes
- post-merge `main` CI is green

Merging Stage 2 still does **not** authorize public-testnet activation or mainnet launch.