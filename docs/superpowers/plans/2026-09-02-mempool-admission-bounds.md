# Mempool Admission Bounds Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove unbounded full-mempool replay from transaction admission while preserving consensus semantics and enforcing explicit mempool/dust/resource limits before mainnet.

**Architecture:** Keep consensus transaction validation unchanged. Make the mempool maintain O(1) aggregate byte accounting plus a sender/nonce index with a conservative per-sender queue cap; admission callers replay only the candidate sender's bounded pending chain. Reorg and persistence rebuild paths use the same sender-scoped view, so unrelated stale entries cannot block valid senders.

**Tech Stack:** Go, existing `blockchain`, `blockchain/mempool`, `p2p`, `transactions`, GitHub Actions security gates.

**Spec:** GitHub issue #104 (`security: bound mempool/transaction resource use and zero-fee dust before mainnet`).

## Global Constraints

- Mainnet launch authorization stays false.
- Mainnet mining authorization stays false.
- Mainnet genesis timestamp stays 0.
- Do not mutate AWS, seeds, keys, or live deployments.
- Preserve existing public-testnet transaction compatibility except for explicit mempool policy tightening.
- Consensus block validation remains independent of local wall-clock rate limiting.

---

### Task 1: Reproduce admission coupling and sender monopolization

**Files:**
- Test: `p2p/mempool_admission_isolation_test.go`
- Test: `blockchain/mempool/mempool_sender_limit_test.go`

**Interfaces:**
- Consumes: current `Node.SubmitTransaction`, `Mempool.AddTransaction`.
- Produces: failing regression evidence for unrelated stale-entry isolation and a per-sender pending bound.

- [x] **Step 1: Write the failing stale-entry isolation test**
- [x] **Step 2: Write the failing per-sender queue-bound test**
- [x] **Step 3: Run CI and verify RED**

Expected RED: the Go pre-audit suite fails on the new regression tests before production changes.

### Task 2: Add constant-time mempool accounting and sender index

**Files:**
- Modify: `params/resource_policy.go`
- Modify: `blockchain/mempool/mempool.go`
- Test: `blockchain/mempool/mempool_sender_limit_test.go`
- Test: `blockchain/mempool/mempool_capacity_test.go`

**Interfaces:**
- Produces: `params.MaxMempoolTransactionsPerSender`, cached `totalEstimatedBytes`, sender/nonce index, and `TransactionsForSender(sender string)`.
- `AddTransaction` must atomically reject duplicate IDs, duplicate sender nonces, global count overflow, byte overflow, and per-sender overflow.
- `RemoveTransaction` and `Clear` must keep all indexes/counters exact.

- [ ] **Step 1: Replace hard-coded sender limit in the RED test with `params.MaxMempoolTransactionsPerSender`**
- [ ] **Step 2: Add `MaxMempoolTransactionsPerSender = 64` to `params/resource_policy.go`**
- [ ] **Step 3: Add cached byte count and sender/nonce index to `Mempool`**
- [ ] **Step 4: Make add/remove/clear maintain counters and indexes under the existing mutex**
- [ ] **Step 5: Add `TransactionsForSender` returning a nonce-ordered copy bounded by the sender cap**
- [ ] **Step 6: Verify mempool tests pass**

### Task 3: Replace full-pool replay in live admission paths

**Files:**
- Modify: `p2p/transaction_submit.go`
- Modify: `p2p/node.go`
- Modify: `p2p/reorg_mempool.go`
- Modify: `p2p/mempool_persistence.go`
- Test: `p2p/mempool_admission_isolation_test.go`
- Test: existing submission/reorg/persistence tests.

**Interfaces:**
- Consumes: `Mempool.TransactionsForSender(tx.From)`.
- Produces: bounded replay for one sender only.

- [ ] **Step 1: Replace `AllTransactions()` with `TransactionsForSender(tx.From)` immediately before `ValidateMempoolTransactionFor` in local submission**
- [ ] **Step 2: Make the peer-originated transaction path use the same sender-scoped pending view**
- [ ] **Step 3: Make reorg recovery validate each candidate against only its sender queue**
- [ ] **Step 4: Make persistence revalidation validate each candidate against only its sender queue**
- [ ] **Step 5: Run p2p tests and confirm stale unrelated entries no longer block valid submissions**

### Task 4: Tighten explicit transaction-size policy and adversarial evidence

**Files:**
- Modify: `params/resource_policy.go`
- Modify: `transactions/resource_bounds.go`
- Test: `transactions/resource_bounds_test.go`
- Test: `blockchain/mempool/mempool_capacity_test.go`

**Interfaces:**
- Produces: a dedicated `MaxTransactionSerializedBytes` distinct from the block byte budget.

- [ ] **Step 1: Add a failing test for an otherwise structurally valid transaction whose estimated serialized size exceeds the dedicated per-transaction maximum**
- [ ] **Step 2: Add `MaxTransactionSerializedBytes = 1024` and enforce it in `ValidateResourceBounds`**
- [ ] **Step 3: Keep `MaxBlockTransactionBytes = 1 MiB` as the aggregate block transaction budget**
- [ ] **Step 4: Verify dust, canonical-address, signature/public-key bounds, per-tx size, global mempool count/bytes, and per-sender limits**

### Task 5: Final verification and documentation

**Files:**
- Update: PR #112 body and issue #104 evidence when green.
- Update audit/remediation docs only if their status text is stale.

- [ ] **Step 1: Run gofmt and `go vet ./...`**
- [ ] **Step 2: Run full Go tests**
- [ ] **Step 3: Run repository-wide race detector**
- [ ] **Step 4: Run security regression/race/adversarial gate**
- [ ] **Step 5: Run two-node rehearsal and public-testnet container build/smoke**
- [ ] **Step 6: Confirm Faucet Recovery CI remains green**
- [ ] **Step 7: Review final PR diff for unrelated changes**
- [ ] **Step 8: Merge only after all gates are green; close #104 with evidence**
