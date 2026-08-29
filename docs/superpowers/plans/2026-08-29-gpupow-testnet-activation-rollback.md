# GPU-PoW v1 Testnet Activation and Rollback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a disabled-by-default, fully rehearsable Version 1 to GPU-PoW Version 2 consensus boundary without arming or deploying it.

**Architecture:** Give each blockchain `Chain` an immutable PoW policy and proof-verification boundary. Keep `NewChain()` legacy-only, build the GPU-capable verifier outside `blockchain` to avoid an import cycle, persist a finite activation decision separately from block/state files, and exercise mixed-version and rollback behavior only in disposable tests.

**Tech Stack:** Go 1.24, existing Sudharma blockchain/pow/P2P/RPC packages, JSON persistence, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-08-29-gpupow-testnet-activation-rollback-design.md`

## Global Constraints

- Testnet and mainnet activation heights remain `params.GPUV1ActivationDisabled` in committed defaults.
- No Seed-1/Seed-2 deployment, AWS/IAM change, public activation, genesis change or monetary change.
- Version 1 is the only valid version while activation is disabled or height is below a finite boundary.
- Version 2 is the only valid version at and after a finite boundary; other versions are always rejected.
- There is no cross-algorithm fallback and no in-chain downgrade after activation.
- CPU verifies Version 2 but never becomes a Version 2 production miner.
- A finite activation must be at least 720 blocks above the tip when first persisted.
- Existing legacy and GPU interoperability vectors remain byte-for-byte unchanged.

---

### Task 1: Immutable chain PoW policy

**Files:**
- Create: `blockchain/pow_policy.go`
- Create: `blockchain/pow_policy_test.go`
- Modify: `pow/gpu_pow_v1_activation.go`

**Interfaces:**
- Produces: `PoWPolicy`, `LegacyOnlyPoWPolicy()`, `VersionAtHeight(uint64) (uint32, error)`, and `VersionAllowed(uint32, uint64) bool`.
- Preserves: `pow.GPUV1VersionAllowedAtHeight` as a compatibility wrapper over the same matrix.

- [ ] **Step 1: Write failing table tests** for disabled, height 99/100/101 around activation 100, versions 0–3, and `^uint64(0)` without arithmetic overflow.
- [ ] **Step 2: Run** `go test ./blockchain ./pow -run 'PoWPolicy|GPUV1VersionAllowed' -count=1`; expect failure because `PoWPolicy` is absent.
- [ ] **Step 3: Implement the immutable value policy** with exact signatures:

```go
type PoWPolicy struct { GPUV1ActivationHeight uint64 }
func LegacyOnlyPoWPolicy() PoWPolicy
func (p PoWPolicy) VersionAllowed(version uint32, height uint64) bool
func (p PoWPolicy) VersionAtHeight(height uint64) (uint32, error)
```

- [ ] **Step 4: Run** `go test ./blockchain ./pow -count=1`; expect PASS.
- [ ] **Step 5: Commit** `feat(gpu-pow): define immutable chain pow policy`.

### Task 2: Chain-owned proof verifier

**Files:**
- Create: `blockchain/proof_verifier.go`
- Create: `blockchain/proof_verifier_test.go`
- Modify: `blockchain/chain.go`
- Modify: `blockchain/validator_core.go`
- Modify: `blockchain/chain_validator.go`
- Modify: `blockchain/validated_chain.go`
- Modify: `blockchain/storage.go`
- Modify: `blockchain/reorg.go`

**Interfaces:**
- Consumes: `PoWPolicy` from Task 1.
- Produces: `ProofVerifier`, `NewChainWithConsensus(PoWPolicy, ProofVerifier) (*Chain, error)`, `Chain.PoWPolicy() PoWPolicy`, and policy-aware replay/reorg.

- [ ] **Step 1: Add failing tests** proving `NewChain()` remains legacy-only, finite activation rejects a nil/legacy-only verifier, wrong versions are rejected before verifier invocation, direct `AddBlock` uses the chain verifier, and candidate/reorg chains preserve policy.
- [ ] **Step 2: Run** `go test ./blockchain -run 'ProofVerifier|ConsensusPolicy' -count=1`; expect missing-interface failures.
- [ ] **Step 3: Define**:

```go
type ProofVerifier interface { Verify(*Block) bool }
type ProofVerifierFunc func(*Block) bool
func (f ProofVerifierFunc) Verify(b *Block) bool { return f(b) }
```

Add immutable `powPolicy` and `proofVerifier` fields to `Chain`. Route `AddBlock` and `ValidateBlockAgainstChain` through `validateBlockCoreWithProof`; retain `ValidateBlockBasic` and `NewChain()` as legacy-only compatibility paths.
- [ ] **Step 4: Ensure storage replay and reorg constructors accept the destination chain policy/verifier** instead of silently calling legacy `NewChain()`.
- [ ] **Step 5: Run** `go test ./blockchain ./p2p ./miner -count=1`; expect PASS.
- [ ] **Step 6: Commit** `feat(gpu-pow): bind consensus proof policy to chain`.

### Task 3: Production GPU verification adapter

**Files:**
- Create: `pow/chain_verifier.go`
- Create: `pow/chain_verifier_test.go`
- Modify: `pow/gpu_pow_v1_reference.go`

**Interfaces:**
- Consumes: `blockchain.ProofVerifier` and the frozen 2 GiB production cache policy.
- Produces: `pow.NewChainProofVerifier(policy blockchain.PoWPolicy) (blockchain.ProofVerifier, error)`.

- [ ] **Step 1: Add failing tests** requiring legacy verification below the boundary, GPU reference verification at/after it, rejection of wrong/future versions, epoch cache selection, and no fallback after a failed Version 2 proof.
- [ ] **Step 2: Run** `go test ./pow -run ChainProofVerifier -count=1`; expect the constructor to be missing.
- [ ] **Step 3: Implement an adapter in `pow`** so the existing import direction remains `pow -> blockchain`; do not import `pow` from `blockchain`. Cache lifecycle must be bounded by epoch and verifier-only.
- [ ] **Step 4: Run** `go test ./pow ./blockchain ./compatibility/gpupowv1 -count=1`; expect PASS and unchanged vectors.
- [ ] **Step 5: Commit** `feat(gpu-pow): add chain verification adapter`.

### Task 4: Candidate and mining-work boundary

**Files:**
- Modify: `blockchain/block_builder.go`
- Modify: `miner/pipeline.go`
- Modify: `rpc/mining_work.go`
- Create: `miner/gpu_activation_test.go`
- Create: `rpc/mining_activation_test.go`

**Interfaces:**
- Produces: policy-selected candidate versions and explicit CPU-miner refusal for Version 2.

- [ ] **Step 1: Add failing tests** showing next-height candidates are Version 1 before activation and Version 2 at activation; CPU `MineNextBlock` returns a stable `external GPU miner required` error for Version 2; mining work is unavailable early and emitted only at/after activation.
- [ ] **Step 2: Run** `go test ./miner ./rpc -run GPUActivation -count=1`; expect failures against the hard-coded Version 1 builder.
- [ ] **Step 3: Add a policy-aware builder** used by consensus mining paths while leaving the old builder as legacy-only compatibility behavior.
- [ ] **Step 4: Gate Version 2 work issuance** on chain policy and reuse the existing immutable work/template binding and host verification.
- [ ] **Step 5: Run** `go test ./miner ./rpc ./blockchain -count=1`; expect PASS.
- [ ] **Step 6: Commit** `feat(gpu-pow): enforce activation mining boundary`.

### Task 5: Persisted activation record and startup validation

**Files:**
- Create: `operations/gpu_activation.go`
- Create: `operations/gpu_activation_test.go`
- Modify: `operations/config.go`
- Modify: `operations/config_test.go`
- Modify: `cmd/sudharma-rpcd/main.go`

**Interfaces:**
- Produces: `GPUV1ActivationHeight` config field, `LoadOrPersistGPUActivation(path string, configured, tip uint64, verifierReady bool)`, and explicit pre-boundary abort validation.

- [ ] **Step 1: Add failing tests** for absent=disabled, finite height with less than 720-block lead rejection, exact 720 acceptance, atomic `0600` record persistence, equal restart acceptance, changed-height rejection, verifier-not-ready rejection, abort below boundary and abort at/after boundary rejection.
- [ ] **Step 2: Run** `go test ./operations -run GPUActivation -count=1`; expect missing API failures.
- [ ] **Step 3: Implement a versioned JSON activation record** in the data directory using temporary-file plus rename persistence. Do not add an automatic clear path.
- [ ] **Step 4: Wire node startup** to construct the policy and verifier before chain replay. Keep the shipped/default configuration disabled and reject finite mainnet activation.
- [ ] **Step 5: Run** `go test ./operations ./cmd/sudharma-rpcd ./... -count=1`; expect PASS.
- [ ] **Step 6: Commit** `feat(gpu-pow): persist unarmed activation policy`.

### Task 6: Status visibility and readiness

**Files:**
- Modify: `rpc/status.go`
- Modify: `rpc/ready.go`
- Modify: `rpc/status_test.go`
- Modify: `rpc/ready_test.go`

**Interfaces:**
- Produces JSON fields: `gpu_pow_phase`, `gpu_v1_activation_height`, `next_block_version`, and `gpu_verifier_ready`.

- [ ] **Step 1: Add failing tests** for disabled, armed and active status plus readiness refusal when a finite activation lacks a ready verifier.
- [ ] **Step 2: Run** `go test ./rpc -run 'GPU.*Status|GPU.*Ready' -count=1`; expect missing fields.
- [ ] **Step 3: Implement deterministic phase derivation** (`disabled`, `armed`, `active`) from chain policy and tip. Never report a disabled sentinel as a plausible date/height.
- [ ] **Step 4: Run** `go test ./rpc -count=1`; expect PASS.
- [ ] **Step 5: Commit** `feat(gpu-pow): expose activation readiness status`.

### Task 7: Disposable mixed-version and rollback rehearsal

**Files:**
- Create: `testnet/gpupow_activation_rehearsal_test.go`
- Create: `operations/gpu_activation_runbook_test.go`
- Create: `docs/testnet/GPU_POW_ACTIVATION_REHEARSAL.md`
- Modify: `.github/workflows/gpu-pow-v1-ci.yml`

**Interfaces:**
- Produces: a non-deployment rehearsal covering boundary, restart, legacy observer, shallow Version 2 reorg, pre-boundary abort and snapshot-only post-boundary recovery.

- [ ] **Step 1: Add a failing documentation contract test** requiring exact safety headings, disabled defaults, 720-block lead, evidence manifest, abort-before-boundary and snapshot-after-boundary language.
- [ ] **Step 2: Add integration tests** with two upgraded chains and one legacy observer. Use small deterministic test verifiers; do not weaken production proof checks.
- [ ] **Step 3: Run** `go test ./testnet ./operations -run 'GPU.*Rehearsal|ActivationRunbook' -count=1`; expect failures until the runbook and fixture exist.
- [ ] **Step 4: Write the operator runbook** with preparation, checksums, disposable paths, exact stop conditions, evidence capture and recovery. State prominently that it does not authorize public seed activation.
- [ ] **Step 5: Add the focused rehearsal to GPU-PoW CI**, then run `go test ./... -count=1` and `go test -race ./blockchain ./pow ./p2p ./rpc ./operations ./testnet -count=1`.
- [ ] **Step 6: Confirm constants remain disabled** with `go test ./pow -run TestGPUV1NetworkActivationDefaultsRemainDisabled -count=1`.
- [ ] **Step 7: Commit** `test(gpu-pow): add activation rollback rehearsal`.

### Task 8: Final evidence and PR checkpoint

**Files:**
- Modify: PR #25 description only after exact-head verification.

- [ ] **Step 1: Run** `git diff --check`, full tests, race tests, node build, secret-safety test, GPU-PoW CI and generic CI.
- [ ] **Step 2: Verify** no activation constant/config is finite, no deployment file changed, no unrestricted CPU Version 2 miner exists, and mainnet remains disabled.
- [ ] **Step 3: Inspect exact-head workflow logs and artifact provenance**; do not treat hosted CI as physical GPU evidence.
- [ ] **Step 4: Update draft PR #25** with exact head, run IDs, remaining RTX 2060/AMD physical gates and the fact that deployment remains unauthorized.
