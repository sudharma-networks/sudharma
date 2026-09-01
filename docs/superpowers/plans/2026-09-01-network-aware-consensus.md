# Network-Aware Consensus Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove public-testnet-only monetary/genesis assumptions from generic chain, peer, miner, persistence and reorg paths before mainnet can be authorized.

**Architecture:** Bind every `blockchain.Chain` to an immutable `params.NetworkID`, derive monetary policy from that identity, and make validation/replay use the chain identity rather than public-testnet compatibility constructors. Preserve `NewChain`, `LoadChainFromFile`, `ProcessBlock` and current public-testnet behavior as compatibility wrappers while adding explicit network-aware entry points for mainnet-capable runtime code.

**Tech Stack:** Go, existing Sudharma `blockchain`, `p2p`, `miner`, `params`, `cmd/sudharmad` packages, GitHub Actions CI/race/rehearsal gates.

**Spec:** GitHub issue #103 and `docs/audits/2026-09-01-internal-security-audit.md`.

## Global Constraints

- Public testnet behavior and persisted testnet files remain compatible.
- `MainnetLaunchAuthorized` remains false.
- `MainnetMiningAuthorized` remains false.
- `MainnetGenesisTimestamp` remains 0.
- No mainnet deployment, seed, AWS, key or secret changes.
- Consensus-critical paths must fail closed on network/policy mismatch.

---

### Task 1: Bind Chain to Network Identity

**Files:**
- Modify: `blockchain/chain.go`
- Test: `blockchain/chain_network_test.go`

**Interfaces:**
- Produces: `func (c *Chain) Network() params.NetworkID`
- `NewChainFor(network)` stores the validated network in the chain.
- `NewChain()` remains a public-testnet compatibility wrapper.

- [ ] **Step 1: Write failing tests** proving `NewChainFor(NetworkMainnet).Network()` is mainnet and clone/replace cannot silently change network identity.
- [ ] **Step 2: Run package tests** and require failure on the baseline implementation.
- [ ] **Step 3: Add immutable `network params.NetworkID` to `Chain`** and a read-only getter.
- [ ] **Step 4: Re-run blockchain tests** and require pass.
- [ ] **Step 5: Commit.**

### Task 2: Make Clone, Validation, Replacement and Reorg Network-Aware

**Files:**
- Modify: `blockchain/reorg.go`
- Modify: `blockchain/validated_chain.go`
- Test: `blockchain/reorg_network_test.go`

**Interfaces:**
- `CloneChain` preserves `source.Network()`.
- `ValidateAndCloneChain` rebuilds from `NewChainFor(source.Network())`.
- `ReplaceWith` rejects a candidate whose network differs from the current chain.
- `BuildStateFromChain` derives monetary policy from `chain.Network()` and creates `NewStateFor(policy)`.

- [ ] **Step 1: Write failing mainnet replay/reorg tests** that use a mainnet chain candidate and mainnet state.
- [ ] **Step 2: Run tests** and require failure from public-testnet genesis/policy assumptions.
- [ ] **Step 3: Implement network-aware clone/validation/replacement/state rebuild.**
- [ ] **Step 4: Run blockchain tests and race tests.**
- [ ] **Step 5: Commit.**

### Task 3: Make Disk Loading Explicit by Network

**Files:**
- Modify: `blockchain/storage.go`
- Modify: `cmd/sudharmad/main.go`
- Test: `blockchain/storage_network_test.go`

**Interfaces:**
- Produces: `LoadChainFromFileFor(path string, network params.NetworkID) (*Chain, error)`.
- `LoadChainFromFile(path)` remains public-testnet compatibility wrapper.
- `cmd/sudharmad` uses `LoadChainFromFileFor(..., network)`.

- [ ] **Step 1: Write failing tests** proving a mainnet chain file can be reloaded only as mainnet and is rejected as public testnet.
- [ ] **Step 2: Run tests and require failure.**
- [ ] **Step 3: Implement `LoadChainFromFileFor` and switch the main daemon call site.**
- [ ] **Step 4: Run storage + daemon tests.**
- [ ] **Step 5: Commit.**

### Task 4: Route P2P Block Acceptance Through Chain Policy

**Files:**
- Modify: `p2p/block_handler.go`
- Test: `p2p/block_handler_network_test.go`

**Interfaces:**
- `Node.AcceptBlock` derives `policy := params.MonetaryPolicyFor(chain.Network())` and calls `ProcessBlockFor`.
- Attached `State.MonetaryPolicy()` must match the chain-derived policy or block acceptance fails closed.

- [ ] **Step 1: Write failing mainnet peer-acceptance test.**
- [ ] **Step 2: Run P2P tests and require failure.**
- [ ] **Step 3: Implement chain-derived policy processing.**
- [ ] **Step 4: Run P2P tests/race tests.**
- [ ] **Step 5: Commit.**

### Task 5: Make Miner Pipeline Explicitly Network-Aware

**Files:**
- Modify: `miner/pipeline.go`
- Test: `miner/network_pipeline_test.go`

**Interfaces:**
- Existing `MineNextBlock` remains compatible but derives policy from `chain.Network()` before state processing.

- [ ] **Step 1: Write failing mainnet mining-policy test.**
- [ ] **Step 2: Run miner tests and require failure.**
- [ ] **Step 3: Replace generic `ProcessBlock` with chain-derived `ProcessBlockFor`.**
- [ ] **Step 4: Run miner tests and focused race tests.**
- [ ] **Step 5: Commit.**

### Task 6: Final Verification and Issue Closure Evidence

**Files:**
- Update: `docs/audits/2026-09-01-internal-security-audit.md`
- Update: GitHub issue #103

- [ ] **Step 1: Run full GitHub Actions CI.**
- [ ] **Step 2: Confirm `go vet`, full Go tests, repository-wide race detector, two-node rehearsal and public-testnet container smoke all pass.**
- [ ] **Step 3: Record final commit and workflow evidence in #103 and audit report.**
- [ ] **Step 4: Close #103 only after verified green evidence.**
- [ ] **Step 5: Keep mainnet authorization/mining/genesis gates unchanged and false/unset.**
