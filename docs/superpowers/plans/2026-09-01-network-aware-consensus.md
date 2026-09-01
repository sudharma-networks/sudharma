# Network-Aware Consensus Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove public-testnet-only monetary/genesis assumptions from generic chain, peer, miner, persistence and reorg paths before mainnet can be authorized.

**Architecture:** Bind every `blockchain.Chain` to an immutable `params.NetworkID`, derive monetary policy from that identity, and make validation/replay use the chain identity rather than public-testnet compatibility constructors. Preserve `NewChain`, `LoadChainFromFile`, `ProcessBlock` and current public-testnet behavior as compatibility wrappers. Mainnet behavior is exercised only through offline validation fixtures/helpers while `MainnetLaunchAuthorized` remains false; no test may authorize or start mainnet.

**Tech Stack:** Go, existing Sudharma `blockchain`, `p2p`, `miner`, `params`, `cmd/sudharmad` packages, GitHub Actions CI/race/rehearsal gates.

**Spec:** GitHub issue #103 and `docs/audits/2026-09-01-internal-security-audit.md`.

## Global Constraints

- Public testnet behavior and persisted testnet files remain compatible.
- `MainnetLaunchAuthorized` remains false.
- `MainnetMiningAuthorized` remains false.
- `MainnetGenesisTimestamp` remains 0.
- No mainnet deployment, seed, AWS, key or secret changes.
- Consensus-critical paths fail closed on network/policy mismatch.
- Offline mainnet validation is allowed only to prove correctness; runtime launch authorization is never bypassed.

---

### Task 1: Bind Chain to Network Identity

**Files:**
- Modify: `blockchain/chain.go`
- Test: `blockchain/chain_network_test.go`

**Interfaces:**
- Produce `func (c *Chain) Network() params.NetworkID`.
- Produce `func (c *Chain) MonetaryPolicy() (params.MonetaryPolicy, error)`.
- `NewChainFor(network)` stores the validated network in the chain and keeps the existing mainnet authorization behavior.
- Test-only/offline validation fixtures construct a mainnet chain from `NewMainnetGenesisBlock()` without changing launch authorization.

- [ ] **Step 1:** Add failing tests for public-testnet identity and an offline mainnet chain fixture.
- [ ] **Step 2:** Verify the tests fail because `Chain` has no network identity/getter.
- [ ] **Step 3:** Add immutable `network params.NetworkID`, getter, policy lookup and an internal validated-genesis constructor.
- [ ] **Step 4:** Re-run `go test ./blockchain -count=1`.
- [ ] **Step 5:** Commit.

### Task 2: Make Clone, Validation, Replacement and Reorg Network-Aware

**Files:**
- Modify: `blockchain/reorg.go`
- Modify: `blockchain/validated_chain.go`
- Modify: `blockchain/fork_choice.go`
- Test: `blockchain/reorg_network_test.go`

**Interfaces:**
- `CloneChain` preserves `source.Network()`.
- `ValidateAndCloneChain` rebuilds from the source network genesis without authorizing runtime mainnet.
- `ReplaceWith` and `BetterChain` reject cross-network candidates.
- `BuildStateFromChain` derives monetary policy from `chain.Network()` and creates `NewStateFor(policy)`.

- [ ] **Step 1:** Add failing cross-network fork/reorg and offline mainnet replay tests.
- [ ] **Step 2:** Verify failure on the baseline public-testnet assumptions.
- [ ] **Step 3:** Implement network-aware clone/validation/replacement/fork-choice/state rebuild.
- [ ] **Step 4:** Run blockchain tests and race tests.
- [ ] **Step 5:** Commit.

### Task 3: Make Disk Loading Explicit by Network

**Files:**
- Modify: `blockchain/storage.go`
- Modify: `cmd/sudharmad/main.go`
- Test: `blockchain/storage_network_test.go`

**Interfaces:**
- Produce `LoadChainFromFileFor(path string, network params.NetworkID) (*Chain, error)` for explicit offline/runtime validation.
- `LoadChainFromFile(path)` remains the public-testnet compatibility wrapper.
- `cmd/sudharmad` uses `LoadChainFromFileFor(..., network)` only after `params.ParseNetwork` has enforced launch authorization.

- [ ] **Step 1:** Add failing tests proving a mainnet-genesis file validates only as mainnet and testnet files remain compatible.
- [ ] **Step 2:** Verify failure.
- [ ] **Step 3:** Implement the network-aware loader and daemon call-site change.
- [ ] **Step 4:** Run storage and daemon tests.
- [ ] **Step 5:** Commit.

### Task 4: Route P2P Block Acceptance Through Chain Policy

**Files:**
- Modify: `p2p/block_handler.go`
- Modify: `p2p/chain_access.go`
- Modify: `p2p/node.go` if required for attachment invariants.
- Test: `p2p/block_handler_network_test.go`

**Interfaces:**
- `Node.AcceptBlock` derives policy from `chain.MonetaryPolicy()` and calls `ProcessBlockFor`.
- `SetChain`/`SetState` fail closed when the attached chain network, P2P namespace and state monetary policy disagree.

- [ ] **Step 1:** Add failing attachment-policy mismatch tests.
- [ ] **Step 2:** Verify failure.
- [ ] **Step 3:** Implement chain-derived block processing and attachment invariants.
- [ ] **Step 4:** Run P2P tests/race tests.
- [ ] **Step 5:** Commit.

### Task 5: Make Miner Pipeline Explicitly Network-Aware

**Files:**
- Modify: `miner/pipeline.go`
- Test: `miner/network_pipeline_test.go`

**Interfaces:**
- `MineNextBlock` derives policy from `chain.MonetaryPolicy()`, verifies the state policy matches, and calls `ProcessBlockFor`.

- [ ] **Step 1:** Add failing chain/state mismatch test.
- [ ] **Step 2:** Verify failure at the intended pre-mining policy gate after implementation.
- [ ] **Step 3:** Replace generic `ProcessBlock` with chain-derived `ProcessBlockFor`.
- [ ] **Step 4:** Run miner tests/race tests.
- [ ] **Step 5:** Commit.

### Task 6: Final Verification and Issue Closure Evidence

**Files:**
- Update: `docs/audits/2026-09-01-internal-security-audit.md`
- Update: GitHub issue #103

- [ ] **Step 1:** Run full GitHub Actions CI.
- [ ] **Step 2:** Confirm formatting, `go vet`, full Go tests, repository-wide race detector, two-node rehearsal and public-testnet container smoke all pass.
- [ ] **Step 3:** Record final commit and workflow evidence in #103 and the audit report.
- [ ] **Step 4:** Close #103 only after verified green evidence.
- [ ] **Step 5:** Re-confirm `MainnetLaunchAuthorized=false`, `MainnetMiningAuthorized=false`, and `MainnetGenesisTimestamp=0`.
