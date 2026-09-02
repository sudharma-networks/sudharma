# Khushi Consensus Reconciliation Stage 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add explicit, immutable, network-aware versioned PoW policy and proof-verifier plumbing to current `main` without changing the public-testnet accepted block set or activating Khushi/mainnet.

**Architecture:** `params` owns disabled activation-height constants. `blockchain` owns an immutable `PoWPolicy` and a `ProofVerifier` interface. `Chain` stores network + policy + verifier and uses them for block admission, clone/reorg, and persistence reconstruction. Existing constructors remain compatibility-safe and select the disabled policy plus legacy SHA256 verifier; new consensus-aware constructors allow a later Stage-2 Khushi verifier to be injected without a package cycle.

**Tech Stack:** Go, GitHub Actions, existing `blockchain`, `params`, `consensus` packages.

**Spec:** `docs/superpowers/specs/2026-09-02-khushi-consensus-reconciliation-design.md`

## Global Constraints

- Start from `c70619af5050cb91deb3ac75eec75962cb428d52` on isolated branch `security/khushi-consensus-reconciliation`.
- `MainnetLaunchAuthorized` remains `false`.
- `MainnetMiningAuthorized` remains `false`.
- `MainnetGenesisTimestamp` remains `0`.
- Both Khushi activation heights remain the disabled sentinel.
- Preserve existing public-testnet Version-1/double-SHA256 consensus behavior.
- Do not add or port CUDA/OpenCL/Khushi Version-2 implementation in Stage 1.
- Do not mutate Seed-1/Seed-2, AWS, IAM, keys, faucet signer state, or public Stratum exposure.
- Do not change monetary policy; mainnet remains exactly 51,000,000 SUDH maximum supply and public testnet remains its legacy 51B development policy.
- Do not flip physical-GPU or public-community-review attestations.

---

### Task 1: Network-aware PoW policy — RED then GREEN

**Files:**
- Create: `blockchain/pow_policy_test.go`
- Create: `blockchain/pow_policy.go`
- Modify: `params/mining.go`

**Interfaces:**
- Produces `blockchain.PoWPolicy{GPUV1ActivationHeight uint64}`.
- Produces `LegacyOnlyPoWPolicy() PoWPolicy`.
- Produces `PoWPolicyForNetwork(network params.NetworkID) (PoWPolicy, error)`.
- Produces `PoWPolicy.VersionAllowed(version uint32, height uint64) bool`.
- Produces `PoWPolicy.VersionAtHeight(height uint64) (uint32, error)`.

- [ ] **Step 1: Write the failing policy tests**

Create `blockchain/pow_policy_test.go` with tests equivalent to:

```go
func TestPoWPolicyForNetworkRemainsDisabled(t *testing.T) {
    for _, network := range []params.NetworkID{params.NetworkPublicTestnet, params.NetworkMainnet} {
        policy, err := PoWPolicyForNetwork(network)
        if err != nil { t.Fatal(err) }
        if policy.GPUV1ActivationHeight != params.GPUV1ActivationDisabled {
            t.Fatalf("network %q activation = %d", network, policy.GPUV1ActivationHeight)
        }
        if !policy.VersionAllowed(1, ^uint64(0)) {
            t.Fatalf("network %q disabled policy rejected Version 1", network)
        }
        if policy.VersionAllowed(2, 1) {
            t.Fatalf("network %q disabled policy accepted Version 2", network)
        }
    }
}
```

Add a table test for a finite policy at height 100: Version 1 allowed at 99, Version 2 rejected at 99, Version 1 rejected at 100, Version 2 allowed at 100/101, versions 0/3 rejected.

Add an unknown-network test requiring `PoWPolicyForNetwork("unknown")` to return an error.

- [ ] **Step 2: Push the test-only commit and verify RED**

Expected GitHub Actions result: `CI` fails at Go compile/test because `PoWPolicy`, `PoWPolicyForNetwork`, or the activation constants do not yet exist. Confirm the failure is caused by the missing Stage-1 API rather than formatting or unrelated tests.

- [ ] **Step 3: Add disabled activation constants**

Append mining-policy constants to `params/mining.go`:

```go
GPUV1ActivationDisabled      uint64 = ^uint64(0)
GPUV1TestnetActivationHeight uint64 = GPUV1ActivationDisabled
GPUV1MainnetActivationHeight uint64 = GPUV1ActivationDisabled
```

Comments must state that Stage 1 is unarmed and any finite height requires a dedicated later consensus activation process.

- [ ] **Step 4: Implement `blockchain/pow_policy.go`**

Implement:

```go
type PoWPolicy struct {
    GPUV1ActivationHeight uint64
}

func LegacyOnlyPoWPolicy() PoWPolicy {
    return PoWPolicy{GPUV1ActivationHeight: params.GPUV1ActivationDisabled}
}

func PoWPolicyForNetwork(network params.NetworkID) (PoWPolicy, error) {
    switch network {
    case params.NetworkPublicTestnet:
        return PoWPolicy{GPUV1ActivationHeight: params.GPUV1TestnetActivationHeight}, nil
    case params.NetworkMainnet:
        return PoWPolicy{GPUV1ActivationHeight: params.GPUV1MainnetActivationHeight}, nil
    default:
        return PoWPolicy{}, fmt.Errorf("unknown network %q", network)
    }
}
```

`VersionAllowed` must explicitly reject versions other than 1/2. `VersionAtHeight` returns 1 when disabled or before a finite activation, otherwise 2.

- [ ] **Step 5: Push and verify GREEN for Task 1**

Expected: policy tests compile/pass; full branch CI should progress beyond these tests. If unrelated CI is still running, do not claim complete until the exact head result is known.

---

### Task 2: Proof-verifier dispatch and Chain integration — RED then GREEN

**Files:**
- Create: `blockchain/proof_verifier.go`
- Create: `blockchain/proof_verifier_test.go`
- Modify: `blockchain/chain.go`
- Modify: `blockchain/validator_core.go`
- Modify: `blockchain/chain_validator.go`

**Interfaces:**
- Consumes `PoWPolicy` from Task 1.
- Produces `ProofVerifier` interface.
- Produces `NewChainForWithConsensus(network params.NetworkID, policy PoWPolicy, verifier ProofVerifier) (*Chain, error)`.
- Produces unexported `newChainFromGenesisForNetworkWithConsensus(...)` used by reconstruction paths.
- Existing `NewChain`, `NewChainFor`, and `newChainFromGenesisForNetwork` remain compatibility wrappers using network policy + legacy verifier.

- [ ] **Step 1: Write failing verifier/chain tests**

Create a `recordingProofVerifier` in `blockchain/proof_verifier_test.go` and require:

```go
func TestNewChainForWithConsensusRequiresPolicyCapabilities(t *testing.T) {
    policy := PoWPolicy{GPUV1ActivationHeight: 100}
    if _, err := NewChainForWithConsensus(params.NetworkPublicTestnet, policy, nil); err == nil {
        t.Fatal("finite activation accepted nil verifier")
    }
    legacyOnly := &recordingProofVerifier{supported: map[uint32]bool{1: true}, result: true}
    if _, err := NewChainForWithConsensus(params.NetworkPublicTestnet, policy, legacyOnly); err == nil {
        t.Fatal("finite activation accepted verifier without Version 2 support")
    }
}
```

Also test that a Version-2 block before activation is rejected before verifier dispatch, and an allowed Version-1 block invokes the configured verifier exactly once.

Add a compatibility test that `NewChainFor(params.NetworkPublicTestnet).PoWPolicy()` equals the disabled network policy.

- [ ] **Step 2: Push test-only commit and verify RED**

Expected: compile/test failure because `ProofVerifier`, `PoWPolicy()` accessor, and consensus-aware constructor do not exist.

- [ ] **Step 3: Implement `ProofVerifier` and legacy verifier**

Create `blockchain/proof_verifier.go`:

```go
type ProofVerifier interface {
    SupportsVersion(version uint32) bool
    Verify(block *Block) bool
}

type legacyProofVerifier struct{}

func (legacyProofVerifier) SupportsVersion(version uint32) bool { return version == 1 }
func (legacyProofVerifier) Verify(block *Block) bool {
    return block != nil && block.Version == 1 && validBlockProofOfWorkCore(block)
}
```

- [ ] **Step 4: Add immutable policy/verifier to `Chain`**

Extend `Chain` with:

```go
powPolicy     PoWPolicy
proofVerifier ProofVerifier
```

Add `PoWPolicy() PoWPolicy` accessor.

Add a constructor validator that rejects nil verifiers, requires Version-1 support, and requires Version-2 support whenever `GPUV1ActivationHeight != params.GPUV1ActivationDisabled`.

Preserve `network`, genesis validation, monetary-policy validation, and cumulative-work initialization from current `main`.

- [ ] **Step 5: Route block validation through policy/verifier**

Refactor `validateBlockCore` into a compatibility wrapper around:

```go
validateBlockCoreWithProof(block, previous, policy, verifier)
```

Order requirements:

1. nil checks
2. policy version check
3. verifier capability check
4. height/parent/timestamp/Merkle/transaction checks
5. verifier proof check

Keep `validBlockProofOfWorkCore` unchanged as the legacy SHA256 rule.

`Chain.AddBlock` and `ValidateBlockAgainstChain` must call `validateBlockCoreWithProof` with the chain's immutable fields.

- [ ] **Step 6: Push and verify GREEN for Task 2**

Expected: new tests pass and existing Version-1 behavior remains green. Confirm no readiness/activation constant changed.

---

### Task 3: Persistence, clone, reorg, and replacement invariants — RED then GREEN

**Files:**
- Modify: `blockchain/reorg.go`
- Modify: `blockchain/validated_chain.go`
- Modify: `blockchain/storage.go`
- Create or modify: `blockchain/reorg_pow_policy_test.go`
- Create or modify: `blockchain/storage_pow_policy_test.go`

**Interfaces:**
- `CloneChain` preserves policy/verifier.
- `ValidateAndCloneChain` rebuilds under source policy/verifier.
- `ReplaceWith` requires candidate network and PoW policy match and revalidates under the current chain verifier.
- `LoadChainFromFileForWithConsensus(path, network, policy, verifier)` is the explicit reconstruction entry point.
- Existing `LoadChainFromFileFor` derives the disabled network policy and legacy verifier.

- [ ] **Step 1: Write failing reconstruction tests**

Add tests that:

- construct a public-testnet chain with finite activation and a recording verifier supporting 1+2, clone it, and assert policy equality
- run `ValidateAndCloneChain` and assert policy equality plus verifier dispatch on admitted blocks
- attempt `ReplaceWith` using a candidate with the same network but a different activation height and require an error containing `proof-of-work policy mismatch`
- save a chain and reload it with `LoadChainFromFileForWithConsensus`, asserting policy equality
- try loading with a finite policy plus legacy-only verifier and require a fail-closed constructor error

- [ ] **Step 2: Push test-only commit and verify RED**

Expected: tests fail because reconstruction paths do not yet preserve/accept explicit consensus configuration.

- [ ] **Step 3: Preserve policy/verifier in clone and validated clone**

`CloneChain` copies `powPolicy` and `proofVerifier`.

`ValidateAndCloneChain` snapshots source network/policy/verifier and calls `newChainFromGenesisForNetworkWithConsensus` before replaying blocks.

- [ ] **Step 4: Harden replacement**

Before candidate revalidation, `ReplaceWith` must reject `c.PoWPolicy() != candidate.PoWPolicy()`.

Revalidation uses `c`'s network, policy, and verifier rather than a compatibility constructor.

Do not replace `c.powPolicy` or `c.proofVerifier` during successful chain replacement; they are immutable runtime configuration.

- [ ] **Step 5: Add consensus-aware storage loading**

Implement:

```go
func LoadChainFromFileForWithConsensus(
    path string,
    network params.NetworkID,
    policy PoWPolicy,
    verifier ProofVerifier,
) (*Chain, error)
```

The existing `LoadChainFromFileFor` resolves `PoWPolicyForNetwork(network)` then calls the new function with `legacyProofVerifier{}`. Therefore a future finite activation cannot silently load with an incapable verifier.

- [ ] **Step 6: Push and verify GREEN for Task 3**

Expected: focused reconstruction tests pass and full CI remains green.

---

### Task 4: Safety assertions, review, and PR

**Files:**
- Modify if needed: `docs/superpowers/specs/2026-09-02-khushi-consensus-reconciliation-design.md`
- Modify if needed: `docs/superpowers/plans/2026-09-02-khushi-consensus-reconciliation.md`
- No activation files should change except adding disabled constants to `params/mining.go`.

**Interfaces:**
- Produces one small Stage-1 PR linked to #118.

- [ ] **Step 1: Verify exact branch diff scope**

Expected changed source files are limited to Stage-1 policy/verifier/reconstruction files and their tests/docs. Reject accidental CUDA/OpenCL, AWS, seed, faucet, Android, tokenomics, or live workflow mutations.

- [ ] **Step 2: Verify hard gates from exact branch head**

Confirm by reading exact-head files:

```text
MainnetLaunchAuthorized = false
MainnetMiningAuthorized = false
MainnetGenesisTimestamp = 0
GPUV1TestnetActivationHeight = GPUV1ActivationDisabled
GPUV1MainnetActivationHeight = GPUV1ActivationDisabled
PhysicalGPUMiningEvidenceComplete = false
PublicCommunitySecurityReviewComplete = false
```

- [ ] **Step 3: Require fresh exact-head CI**

Require the complete `CI` workflow to finish `success` on the final Stage-1 branch head. Do not rely on a prior commit's green run.

- [ ] **Step 4: Open a focused draft PR**

Title:

`security: add fail-closed versioned PoW consensus plumbing`

PR body must state:

- closes only Stage 1 of #118
- accepted public-testnet proof behavior is unchanged
- Khushi Version-2 verification is not yet wired
- activation heights remain disabled
- mainnet remains NO-GO
- no AWS/seed/key/public-Stratum changes
- internal AI-assisted readiness work is not an independent audit

- [ ] **Step 5: Verify PR exact head and diff**

Re-read PR head SHA, changed filenames, mergeability, and current `main` SHA. Do not merge if `main` unexpectedly advanced without reviewing/rebasing the scope.

- [ ] **Step 6: Merge only after all checks are green and exact-head verification passes**

Use expected-head protection when the connector supports it. After merge, verify actual `main` and post-merge CI before claiming Stage 1 complete.
