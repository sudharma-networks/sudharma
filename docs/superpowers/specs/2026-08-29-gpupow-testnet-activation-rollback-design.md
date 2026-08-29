# GPU-PoW v1 Testnet Activation and Rollback Design

## Status and scope

This is a pre-deployment consensus design for the Sudharma public testnet. It
does not arm an activation height, change either seed configuration, deploy a
binary, enable unrestricted mining, or alter mainnet. The existing sentinels
`GPUV1TestnetActivationHeight` and `GPUV1MainnetActivationHeight` remain
disabled while this design and its tests are developed.

The purpose is to make the eventual Version 1 to Version 2 boundary explicit,
rehearsable, observable, and recoverable before the boundary. Physical NVIDIA
and AMD/OpenCL evidence remains a prerequisite for any live activation.

## Existing constraints

- Legacy blocks use header Version 1 and the current double-SHA-256 proof.
- Khushi Algorithm GPU-PoW v1 uses header Version 2 and the frozen
  `sudharma-gpupow-v1` contract.
- Before activation, only Version 1 is valid.
- At and after activation, only Version 2 is valid.
- Mainnet GPU-PoW remains disabled without a separate design and approval.
- Nodes verify Version 2 proof independently on CPU. CPU production mining
  fallback remains prohibited.
- Monetary policy, genesis, difficulty, fees, subsidy, treasury, wallet and
  transaction semantics do not change.
- No automatic rule may make a Version 1 block valid again after an activated
  Version 2 boundary.

## Considered approaches

### A. Explicit height with a chain-owned validation policy — selected

Every node receives the same network policy containing an activation height.
The chain admission path selects the permitted header version and proof
verifier from that policy. Candidate construction uses the same policy. This
keeps the boundary deterministic while allowing a disabled rehearsal and a
future-height activation without a second consensus implementation.

### B. Compile-time activation constant

This gives identical behavior for identical binaries, but changing or
aborting the height requires rebuilding and redeploying. It makes operator
rehearsal and pre-boundary aborts unnecessarily slow and error-prone.

### C. Signalling or miner-vote activation

This adds voting windows, threshold state and additional fork cases. Sudharma
currently has two controlled public-testnet seeds, so signalling provides no
benefit proportional to its consensus complexity.

## Consensus policy

Define an immutable `PoWPolicy` when a chain is opened:

```go
type PoWPolicy struct {
    GPUV1ActivationHeight uint64
}
```

The disabled value remains `params.GPUV1ActivationDisabled`. The policy is
validated once at startup and cannot be mutated while the process is running.
All block admission paths, including direct `Chain.AddBlock`, P2P admission,
local mining and replay from storage, use the chain-owned policy.

For block height `h` and activation height `a`:

| State | Version 1 | Version 2 | Other versions |
|---|---:|---:|---:|
| `a` disabled | accept | reject | reject |
| `h < a` | accept | reject | reject |
| `h >= a` | reject | accept | reject |

Version selection happens before proof verification. A wrong-version block is
rejected without attempting the other algorithm. Difficulty and accumulated
work continue to use the existing Sudharma rules.

## Validation architecture

The current `blockchain` package verifies `Block.Hash()`, which is the legacy
proof, while the `pow` package imports `blockchain`. Importing `pow` back into
`blockchain` would create a cycle. The activation work therefore introduces a
small proof-verification boundary in `blockchain`:

```go
type ProofVerifier interface {
    Verify(block *Block) bool
}
```

`Chain` owns an immutable verifier and `PoWPolicy`. The default constructor
retains a legacy-only verifier so existing callers remain safe. A network
constructor must explicitly supply a GPU-capable verifier before a finite
activation height is accepted. `ValidateBlockBasic` remains legacy-only for
compatibility tests; consensus admission uses the policy-aware chain method.

The GPU-capable verifier is assembled outside `blockchain`, where importing
both `blockchain` and `pow` does not create a cycle. It dispatches Version 1 to
the frozen legacy verifier and Version 2 to the frozen GPU-PoW reference
verifier with the production epoch-cache policy. It never falls back from one
algorithm to the other.

Candidate construction asks the chain policy for the required version at the
next height. The existing CPU miner refuses Version 2 candidates. External GPU
mining work is issued only for Version 2 at or after the boundary.

## Configuration and startup safeguards

The testnet node configuration may eventually expose one field:

```json
{
  "gpu_v1_activation_height": 18446744073709551615
}
```

The disabled sentinel is the default when the field is absent. A finite value
is accepted only when all of these checks pass:

1. the node was built with GPU-PoW v1 verification support;
2. the configured network is the Sudharma public testnet;
3. the height is at least 720 blocks above the persisted tip at the time the
   activation configuration is first armed;
4. the persisted activation record is absent or exactly equal to the supplied
   value; and
5. mainnet activation remains disabled.

At the 60-second target, 720 blocks provide approximately twelve hours for
both nodes and operators to detect disagreement. Once a finite activation
height is accepted, the node stores it beside chain metadata. A later startup
cannot silently change it. Before activation, an explicit abort operation may
clear the stored value only when the chain tip is below the activation height
and both nodes are stopped. This operation is never automatic.

Each node exposes the configured and persisted activation height, current
phase (`disabled`, `armed`, or `active`), required next block version and GPU
verifier readiness in status and readiness output.

## Mixed-version behavior

An old legacy-only node and an armed upgraded node agree below the boundary.
At the boundary the old node cannot validate Version 2 and must stop following
the upgraded chain. This is expected and must be demonstrated in rehearsal;
the project must not describe old nodes as compatible after activation.

Before arming, both public seeds must run the same GPU-capable binary with the
activation disabled. Their binary checksum, production policy, canonical
vectors and status output must match. Both are then stopped, configured with
the same future height, restarted, and checked for identical persisted policy.

## Rehearsal sequence

The controlled two-node rehearsal uses disposable data directories and no
public seed deployment:

1. start two upgraded nodes with activation disabled and verify Version 1;
2. arm both at a future local height and verify identical status;
3. keep one legacy-only observer below the boundary;
4. mine the last Version 1 block and reject an early Version 2 block;
5. submit the first independently verified Version 2 GPU block at the exact
   boundary;
6. reject Version 1 at and after the boundary;
7. restart both upgraded nodes and replay the active chain;
8. demonstrate the legacy observer stopping at the boundary;
9. test shallow Version 2 reorganization within the existing 120-block limit;
10. retain logs, status snapshots, blocks, binary hashes and a SHA-256 evidence
    manifest.

No rehearsal step sends test coins, changes public seed state or treats hosted
CI as physical GPU evidence.

## Abort and rollback rules

### Before activation

Operators may abort only while every participating node tip is below the
activation height. Stop both nodes, preserve data snapshots and evidence,
clear the identical persisted activation record through the explicit abort
procedure, restore disabled configuration, restart, and verify Version 1-only
status. If either node has reached the boundary, this procedure is forbidden.

### At or after activation

There is no in-chain downgrade. Nodes never reinterpret Version 2 history or
resume Version 1 at a later height. Recovery from a failed rehearsal requires
stopping every participating node and restoring the same pre-activation data
snapshot and disabled configuration. Public-testnet use additionally requires
a documented incident decision; it is not automated by node software.

Normal shallow Version 2 reorganizations remain governed by
`MaxAutomaticReorgDepth`. They do not cross the activation boundary into a
Version 1 continuation.

## Test requirements

Automated tests must cover:

- disabled, pre-boundary, exact-boundary and post-boundary version matrices;
- wrong-version and future-version rejection before proof dispatch;
- no cross-algorithm fallback;
- direct `Chain.AddBlock`, P2P and local admission using the same policy;
- legacy constructor behavior remaining Version 1-only;
- candidate version selection and CPU-miner refusal after activation;
- missing GPU verifier rejection for a finite activation height;
- 720-block arming lead-time validation;
- persisted policy equality on restart and rejection of silent changes;
- abort allowed only below the boundary;
- replay, restart, mixed-version observer and bounded Version 2 reorganization;
- mainnet remaining disabled; and
- existing legacy vectors and GPU interoperability vectors remaining unchanged.

## Live activation prerequisites

Implementation and rehearsal tests do not authorize deployment. Before the
public seeds can be armed, all of the following evidence must exist:

1. packaged RTX 2060 localhost staging round trip accepted by the independent
   Go verifier;
2. equivalent physical AMD or other non-NVIDIA OpenCL 4 GiB+ result;
3. independent cross-vendor evidence review;
4. complete disposable two-node activation and recovery rehearsal;
5. identical node binary and configuration checksums; and
6. a separate explicit authorization naming the public activation height.

Mainnet requires a separate proposal and remains disabled regardless of the
testnet result.
