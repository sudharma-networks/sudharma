# KAWPOW-Derived GPU-PoW v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the temporary Version-2 hashing scaffold with a Sudharma-specific, KAWPOW/ProgPoW-derived GPU-oriented proof-of-work and deploy it safely to the public testnet after deterministic cross-implementation verification.

**Architecture:** Preserve Sudharma block/header, monetary, wallet, difficulty and supply semantics while adopting proven KAWPOW/ProgPoW concepts: epoch cache/DAG, GPU-friendly random memory access, programmatic 32-bit mixing and deterministic light verification. Nodes verify on CPU; supported production mining is external GPU mining, CUDA first and AMD/OpenCL second. Activation is explicitly versioned and gated on two-node testnet interoperability.

**Tech Stack:** Go consensus verifier, CUDA C/C++ NVIDIA miner, later OpenCL/AMD, GitHub Actions, Sudharma RPC/P2P, AWS public testnet.

**Spec:** `docs/superpowers/specs/2026-08-27-gpu-pow-v1-design.md`

## Global Constraints
- Do not copy another coin's network identifiers, genesis data, rewards, addresses, difficulty constants or chain-specific personalization.
- Preserve Sudharma 60-second target, 50 SUDH initial subsidy, 1,000,000-block halving interval, 100,000,000 SUDH max supply and zero premine.
- Legacy Version-1 blocks remain byte-for-byte compatible.
- GPU-PoW Version-2 must be deterministic and domain-separated from legacy PoW.
- CPU implementation is consensus verifier/reference only, not a supported production mining mode.
- Do not deploy consensus changes to Seed-1/Seed-2 until vectors, regression tests and CUDA interoperability pass.
- No arbitrary minting or balance edits.

---

### Task 1: Freeze the Version-2 algorithm contract
- [ ] Add failing tests for algorithm ID, domain separation, epoch derivation, seed derivation and canonical header input.
- [ ] Run focused tests and record the intended RED failure.
- [ ] Implement constants/types only.
- [ ] Run focused tests to GREEN.
- [ ] Commit.

### Task 2: Implement deterministic epoch cache generation
- [ ] Add small deterministic cache vectors suitable for CI.
- [ ] Implement seed/hash expansion with explicit byte order and bounds.
- [ ] Add malformed/overflow tests.
- [ ] Run `go test ./pow -count=1`.
- [ ] Commit.

### Task 3: Implement dataset-item generation and light verifier
- [ ] Add fixed dataset-item vectors for multiple epochs/indexes.
- [ ] Implement deterministic item derivation from cache.
- [ ] Add tests proving verifier memory is bounded and output stable.
- [ ] Run focused and regression tests.
- [ ] Commit.

### Task 4: Implement programmatic GPU-oriented mix
- [ ] Define Sudharma-specific period/program seed schedule.
- [ ] Add fixed instruction/mix vectors covering integer math, rotations, merge operations and DAG reads.
- [ ] Implement canonical reference evaluator in Go.
- [ ] Verify byte-for-byte vectors.
- [ ] Commit.

### Task 5: Compose final GPU-PoW v1 digest and target check
- [ ] Add end-to-end header+nonce+mix+final-digest vectors.
- [ ] Replace temporary domain-separated double-SHA scaffold behind Version 2.
- [ ] Keep Version 1 on legacy `HashBlock`.
- [ ] Update `CheckBlock` to enforce Version-appropriate semantics.
- [ ] Run `go test ./pow ./blockchain ./consensus -count=1`.
- [ ] Commit.

### Task 6: Add activation-height consensus rules
- [ ] Add pre/post activation acceptance/rejection tests.
- [ ] Add explicit testnet activation parameter; mainnet remains disabled.
- [ ] Reject wrong-version blocks across activation boundary.
- [ ] Run chain/reorg/regression tests.
- [ ] Commit.

### Task 7: Implement constrained external mining work API
- [ ] Add tests for immutable work template fields, work ID, target, algorithm/version and reward address binding.
- [ ] Add solution submission tests for valid, invalid, stale and mutated work.
- [ ] Implement get-work/submit-work without exposing administrative RPC.
- [ ] Add rate-limit-ready boundaries and telemetry counters.
- [ ] Commit.

### Task 8: Implement NVIDIA CUDA miner
- [ ] Port only the generic GPU execution concepts needed by the approved algorithm; preserve Sudharma-specific consensus constants and vectors.
- [ ] Make CUDA implementation pass the same fixed vectors as Go.
- [ ] Implement GPU DAG generation/search, nonce-range dispatch and stale-work cancellation.
- [ ] Add hashrate, accepted/rejected, temperature, power and utilization reporting where NVIDIA tooling exposes them.
- [ ] Prohibit silent CPU mining fallback.
- [ ] Commit.

### Task 9: Build Windows RTX 2060 artifact in CI
- [ ] Add Windows/CUDA build workflow with source revision and checksum metadata.
- [ ] Produce a test artifact suitable for the office RTX 2060 PC.
- [ ] Add benchmark mode that does not submit blocks.
- [ ] Verify artifact provenance/checksum.
- [ ] Commit.

### Task 10: RTX 2060 interoperability gate
- [ ] Detect GPU/driver/CUDA capability.
- [ ] Run canonical vector self-test on the RTX 2060.
- [ ] Run fixed-duration benchmark and record H/s, watts, temperature and hashes/watt.
- [ ] Submit a controlled test solution to a non-production/staging endpoint.
- [ ] Do not proceed if GPU and Go verifier disagree.

### Task 11: Seed-1/Seed-2 staged testnet activation
- [ ] Back up node configs/state metadata and record current tip/height/work/supply.
- [ ] Deploy identical GPU-PoW-capable node binaries with activation still disabled.
- [ ] Verify Seed-1/Seed-2 remain synchronized.
- [ ] Configure explicit future activation height.
- [ ] Mine first Version-2 block externally on GPU.
- [ ] Confirm both nodes independently accept identical tip/work/supply.
- [ ] Verify invalid Version-2 solution rejection.

### Task 12: Faucet and Android end-to-end test
- [ ] Direct legitimately mined reward to controlled mining address or transfer it normally to faucet.
- [ ] Confirm spendable faucet reserve.
- [ ] Request exactly 100 Test SUDH from Android wallet.
- [ ] Confirm transaction, wallet refresh and faucet reserve accounting.
- [ ] Verify supply increased only through block subsidy.

### Task 13: AMD/OpenCL portability and mining protocol compatibility
- [ ] Implement AMD/OpenCL worker against unchanged vectors.
- [ ] Add Stratum-compatible job translation without changing consensus semantics.
- [ ] Benchmark second vendor and document efficiency.
- [ ] Run full cross-vendor regression suite.

## Release Gate
GPU-PoW v1 is not deployable until Go reference vectors pass, CUDA matches every canonical vector, no CPU production fallback exists, Seed-1 and Seed-2 independently validate a real GPU-mined Version-2 block, and the mined-funds -> faucet -> Android 100 SUDH path succeeds without arbitrary issuance.
