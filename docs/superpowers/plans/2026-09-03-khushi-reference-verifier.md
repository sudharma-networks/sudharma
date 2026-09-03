# Khushi Reference Verifier Reconciliation Implementation Plan

> Execute with strict RED -> intended failure -> minimal GREEN -> exact-head verification. Never enable GPU-PoW activation, mainnet launch/mining, public Stratum or live infrastructure in this plan.

**Goal:** Reconcile the deterministic `sudharma-gpupow-v1` CPU reference verifier and canonical fixtures onto current main, then prove it works through the Stage-1 consensus interfaces under an explicit finite test policy while default network activation remains disabled.

**Base:** `c523ff7a407b8ad442f15d175dfddb6bf9086b00`

**Spec:** `docs/superpowers/specs/2026-09-03-khushi-reference-verifier-design.md`

## Global invariants

- Mainnet max supply remains exactly 51,000,000 SUDH; no monetary files change.
- `MainnetLaunchAuthorized=false`.
- `MainnetMiningAuthorized=false`.
- `MainnetGenesisTimestamp=0`.
- Both GPU activation heights remain `GPUV1ActivationDisabled`.
- Default network remains public testnet.
- Physical GPU and public/community review gates remain false.
- Version-1 PoW code remains unchanged.
- No AWS, seeds, keys, wallet, faucet, deployment, public RPC or Stratum mutation.
- No `pow` dependency on `compatibility`.
- CPU implementation verifies only; it is not a supported production miner.

---

## Task 1 — Freeze contract, epoch and production verifier cache constants

**Files**
- Add: `pow/gpu_pow_v1_contract_test.go`
- Add: `pow/gpu_pow_v1_contract.go`

**RED**
- Assert algorithm ID is exactly `sudharma-gpupow-v1` and matches `params.ProductionMiningAlgorithm`.
- Assert epoch length 7,500 and boundary derivation.
- Assert deterministic epoch seeds for epochs 0, 1 and a nontrivial epoch.
- Assert production verifier cache bytes = 16 MiB, node bytes = 64, node count = 262,144.
- Assert no alternate activation value is introduced.

Run exact-head CI and confirm missing symbols / contract failures are the intended RED.

**GREEN**
- Add constants and epoch seed implementation only.
- Keep hardware-only dataset/reserve/VRAM constants out of this file.
- Verify focused tests, full CI.

---

## Task 2 — Reconcile cache and dataset-item reference primitives

**Files**
- Add: `pow/gpu_pow_v1_cache_test.go`
- Add: `pow/gpu_pow_v1_dataset_test.go`
- Add: `pow/gpu_pow_v1_cache.go`
- Add: `pow/gpu_pow_v1_dataset.go`

**RED**
- Port compact deterministic cache vectors from reviewed PR #25.
- Port dataset-item vectors for multiple indexes/epochs.
- Add zero-node / empty-cache fail-closed tests.

**GREEN**
- Port deterministic Keccak-512 cache generation.
- Port dataset-item derivation with explicit little-endian word handling and fixed parent rounds.
- No production miner code.

Verify `go test ./pow -count=1` via CI and full repository gates.

---

## Task 3 — Reconcile programmatic mix and final digest

**Files**
- Add tests for RNG, permutations, lane initialization, math/merge, program loop, reduction, final digest.
- Add:
  - `pow/gpu_pow_v1_rng.go`
  - `pow/gpu_pow_v1_mix.go`
  - `pow/gpu_pow_v1_permutation.go`
  - `pow/gpu_pow_v1_lane.go`
  - `pow/gpu_pow_v1_program_loop.go`
  - `pow/gpu_pow_v1_group.go`
  - `pow/gpu_pow_v1_reduce.go`
  - `pow/gpu_pow_v1_final.go`

**RED**
- Fixed seed/RNG vectors.
- Register permutation vectors.
- Random math/merge edge vectors.
- Program period boundary tests.
- Group reduction/final digest vectors.

**GREEN**
- Port only the reviewed deterministic arithmetic/reference primitives.
- Preserve uint32 wraparound and explicit endianness.

Verify focused + race tests.

---

## Task 4 — Canonical header/nonce and interoperability vectors

**Files**
- Add: `pow/gpu_pow_v1_reference.go`
- Add: `pow/gpu_pow_v1_reference_test.go`
- Add: `pow/gpu_pow_v1_block_interop_test.go`
- Add: `docs/gpu-pow-v1-interoperability-vectors.json`

**RED**
- Load the canonical compact fixture.
- Assert all six reviewed compact reference digests.
- Assert canonical Version-2 block header prefix and digest.
- Add mutations of nonce/header/miner/previous hash/merkle/timestamp/height/difficulty that must not reproduce the canonical digest.

**GREEN**
- Add canonical `domain || header-prefix || nonce-little-endian` reference digest.
- Bind the program schedule and epoch cache.
- Add block reference/hash/check helpers that do not alter legacy `HashBlock` or `CheckBlock`.
- Fixture algorithm ID must be `sudharma-gpupow-v1`.

---

## Task 5 — Add the Khushi-capable chain proof verifier

**Files**
- Add: `pow/chain_verifier.go`
- Add: `pow/chain_verifier_test.go`

**RED**
- Version 1 accepted under pre-activation finite policy using legacy proof.
- Version 2 rejected before activation.
- Version 1 rejected at/after activation.
- Version 2 dispatched at/after activation.
- unknown version rejected.
- incapable / invalid inputs fail closed.
- cache rebuild occurs only across epoch changes.
- concurrent verification/cache access is race-safe.

**GREEN**
- `NewChainProofVerifier(policy)` uses fixed production verifier cache node count.
- package-private constructor allows compact cache count in tests.
- one-epoch lazy cache protected by mutex.
- no CPU mining API/fallback.

---

## Task 6 — Explicit finite-policy blockchain integration and replay negatives

**Files**
- Add focused integration tests under `pow` or `blockchain` without creating an import cycle.
- Modify only if required: node-independent composition helpers; do not change default network activation.

**RED**
- Build `NewChainForWithConsensus` using an explicit finite policy + `NewChainProofVerifier`.
- prove valid Version-2 proof can pass at the activation boundary using a compact/testing verifier where practical.
- prove wrong nonce / header mutation / pre-activation replay / Version-1 post-activation rejection.
- prove compatibility/default network constructors still expose activation-disabled policy and legacy verifier behavior.

**GREEN**
- Add the minimal composition required by tests.
- Do not wire default node startup in this reference-verifier PR if doing so would broaden scope; open the next focused Stage-2 integration PR from the merged result.

---

## Task 7 — Exact-head review and merge gate

1. Verify branch diff contains only `pow`, canonical fixture, tests, spec/plan and any minimal consensus integration test.
2. Re-read exact-head:
   - `params/network.go`
   - `params/mining.go`
   - `params/security_review_evidence.go`
3. Require push CI success including:
   - formatting/vet
   - `go test ./...`
   - repository-wide race detector
   - security regression/race/adversarial gate
   - local two-node rehearsal
   - public-testnet container build/smoke
4. Open focused draft PR against exact current `main`.
5. State explicitly that this is internal / AI-assisted readiness, not independent audit/certification.
6. Require PR CI + Faucet Recovery CI green on exact head.
7. Merge only with expected-head SHA protection.
8. Require post-merge main CI green before starting the next Stage-2 node/runtime integration slice.

## Completion meaning

Completing this plan means current `main` has a deterministic, independently testable Khushi CPU reference verifier behind explicit consensus policy. It does **not** mean GPU-PoW is activated, physical hardware evidence is complete, public review is complete, or mainnet is authorized/live.