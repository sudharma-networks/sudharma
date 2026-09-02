# Khushi Consensus Reconciliation Design

## Classification

This is a consensus-critical, pre-mainnet design for a maintainer-controlled, AI-assisted internal readiness process. It is not an independent third-party security audit, certification, or launch authorization.

## Baseline

Stage 1 starts from exact `main` commit:

`c70619af5050cb91deb3ac75eec75962cb428d52`

Tracking issue: #118.

Current `main` advertises `sudharma-gpupow-v1` / Khushi as the production GPU-only mining algorithm, while `blockchain.validBlockProofOfWorkCore` still validates the canonical double-SHA256 `Block.Hash()` against the difficulty target. Draft PR #25 contains an older versioned PoW policy and Khushi verifier implementation, but that PR is heavily diverged and must not be merged wholesale.

## Goal

Reconcile the intended versioned proof-of-work architecture onto current network-aware `main` in small reviewable stages, while preserving the live public-testnet chain and keeping all GPU-PoW/mainnet activation gates fail-closed.

## Hard safety constraints

The following values and operational boundaries must not change in this work:

- `MainnetLaunchAuthorized = false`
- `MainnetMiningAuthorized = false`
- `MainnetGenesisTimestamp = 0`
- default network remains `sudharma-testnet-1`
- mainnet network remains `sudharma-mainnet-1`
- no finite public-testnet or mainnet Khushi activation height is introduced in Stage 1
- no Seed-1/Seed-2 mutation
- no AWS/IAM/key mutation
- no public Stratum listener exposure
- no physical GPU evidence is inferred, simulated, or marked complete
- the existing public-testnet genesis/history is not rewritten
- the final mainnet monetary policy remains exactly 51,000,000 SUDH maximum supply

## Staged architecture

### Stage 1 — immutable PoW policy and verifier plumbing

Stage 1 is deliberately behavior-preserving. It adds the consensus structure needed to distinguish legacy Version-1 proof validation from future Khushi Version-2 validation, but both networks remain legacy-only because their Khushi activation heights are the explicit disabled sentinel.

Add three mining-policy constants in `params`:

- `GPUV1ActivationDisabled = ^uint64(0)`
- `GPUV1TestnetActivationHeight = GPUV1ActivationDisabled`
- `GPUV1MainnetActivationHeight = GPUV1ActivationDisabled`

Add `blockchain.PoWPolicy` with one immutable field, `GPUV1ActivationHeight uint64`, plus:

- `LegacyOnlyPoWPolicy()`
- `PoWPolicyForNetwork(params.NetworkID)`
- `VersionAllowed(version, height)`
- `VersionAtHeight(height)`

The disabled sentinel always selects Version 1. A finite activation height selects Version 1 strictly before the height and Version 2 at/after it. Versions other than 1 or 2 are rejected.

Add `blockchain.ProofVerifier`:

```go
type ProofVerifier interface {
    SupportsVersion(version uint32) bool
    Verify(block *Block) bool
}
```

The Stage-1 concrete verifier is legacy-only and delegates Version-1 validation to the existing double-SHA256 proof check. It does not implement Khushi.

`Chain` stores its immutable `PoWPolicy` and `ProofVerifier` alongside its immutable `NetworkID`. Construction validates that the verifier supports every block version the configured policy can select. A finite GPU-v1 activation with a legacy-only verifier must fail closed during construction.

Existing public constructors keep their current signatures and behavior. New consensus-aware constructors are added so Stage 2 can inject a Khushi-capable verifier without making `blockchain` import the `pow` package and creating a dependency cycle.

### Stage 1 persistence/reorg invariants

PoW policy and verifier are runtime consensus configuration, not serialized chain data. They must be preserved or explicitly supplied whenever a chain is reconstructed:

- `CloneChain` copies network, policy, and verifier.
- `ValidateAndCloneChain` rebuilds using the source network, policy, and verifier.
- `ReplaceWith` revalidates a candidate under the current chain's policy/verifier and rejects policy mismatch.
- storage loading gets a consensus-aware entry point that requires network + policy + verifier; the existing compatibility loader derives the disabled network policy and legacy verifier.
- the default/zero value of `PoWPolicy` must never be silently treated as a valid runtime policy. Constructors always set it explicitly.

This prevents a zero-valued activation height from accidentally meaning "activate Version 2 at genesis."

### Stage 2 — deterministic Khushi Version-2 verifier

Stage 2 is a separate PR after Stage 1 is merged and reverified. It ports only the deterministic/reference Khushi pieces required for independent CPU verification and frozen vectors from PR #25 into a fresh branch based on then-current `main`.

Stage 2 wires a verifier that supports Version 1 and Version 2 into runtime construction paths, while keeping both activation heights at `GPUV1ActivationDisabled`. Therefore Stage 2 also must not alter the currently accepted public-testnet block set.

CUDA/OpenCL mining binaries remain external producers of nonces; consensus nodes independently verify any future Version-2 block using deterministic host-side verification. CPU production mining remains unsupported.

### Stage 3 — exact-revision physical interoperability evidence

Only after the reconciled verifier/vector implementation is merged and stable should CUDA/OpenCL localhost-staging packages be rebuilt from that exact revision. Real physical evidence remains required in #24:

- RTX 2060 localhost staging acceptance ending `local-staging-gate=accepted`
- physical AMD/non-NVIDIA OpenCL GPU with at least 4 GiB dedicated VRAM passing memory/vector/benchmark and localhost staging acceptance

Passing localhost staging does not activate consensus or create a public block.

### Stage 4 — public/community security review

Do not start the 14-day public/community review window while Stage 1/Stage 2 consensus-critical changes are pending. After the exact review candidate is stable, pin one commit, announce the review using the existing procedure, and restart/re-pin if material consensus/security changes land during the window.

## Stage-1 consensus flow

For an incoming block:

1. The chain computes expected history-based difficulty exactly as today.
2. The chain's immutable `PoWPolicy` checks whether the block version is permitted at that height.
3. The chain verifies that its injected `ProofVerifier` supports the selected version.
4. Normal height/parent/timestamp/Merkle/transaction checks run.
5. The injected verifier validates proof of work.
6. Only after all validation passes may the block be appended and cumulative work updated.

With both activation heights disabled, only Version 1 reaches step 5 and the existing double-SHA256 proof rule remains authoritative.

## Failure behavior

- Unknown network: construction fails.
- Unknown block version: block rejected before proof verification.
- Finite Version-2 activation + verifier without Version-2 support: chain construction fails.
- Candidate chain with different network or different PoW policy: replacement/reorg rejected.
- Stored chain loaded without a verifier capable of its configured active version: load fails.
- Mainnet construction remains subject to existing launch gates for runtime constructors; offline review helpers remain non-launching.

## Testing strategy

Stage 1 follows RED → GREEN TDD through GitHub Actions on the isolated `security/khushi-consensus-reconciliation` branch.

Required tests:

1. disabled policy accepts Version 1 and rejects Version 2 at representative and maximum heights
2. finite policy selects Version 1 before activation and Version 2 at/after activation
3. unsupported/future versions are rejected
4. finite activation cannot construct a chain with a legacy-only verifier
5. wrong block version is rejected before verifier dispatch
6. configured verifier is actually invoked for an allowed version
7. clone/validated-clone preserve network + policy + verifier behavior
8. replacement rejects mismatched PoW policy
9. storage consensus-aware load preserves explicit policy and fails closed with insufficient verifier capability
10. all existing network-aware, reorg, persistence, security/race, two-node, container, and readiness suites remain green

## Review boundary for the first PR

The first PR closes only Stage 1 of #118. It must not import the old PR #25 tree wholesale, must not add CUDA/OpenCL implementation files, and must not flip any readiness attestation. Its purpose is to make the consensus boundary explicit and fail-closed while preserving the exact currently accepted public-testnet proof behavior.
