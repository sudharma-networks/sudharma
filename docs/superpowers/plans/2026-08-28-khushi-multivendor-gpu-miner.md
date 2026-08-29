# Khushi Algorithm Multi-Vendor GPU Miner Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Finish Sudharma GPU-PoW v1 external mining with a real NVIDIA CUDA search path, a portable OpenCL backend, capability-derived memory checks, Windows artifacts and a reproducible hardware interoperability procedure.

**Architecture:** Keep the frozen `sudharma-gpupow-v1` consensus contract and vectors unchanged. Reuse one shared mining-work/nonce/stale/target contract across CUDA and OpenCL; vendor code owns only device discovery, allocation, kernel execution and optional telemetry. CPU remains verifier/reference only and must never become a production mining fallback.

**Tech Stack:** Go tests/reference verifier, CUDA C++/NVIDIA runtime, OpenCL C, GitHub Actions, Windows PowerShell, Sudharma external mining API.

**Spec:** `docs/superpowers/specs/2026-08-28-khushi-multivendor-gpu-miner-design.md`

## Global Constraints

- Human-facing name is `Khushi Algorithm`; consensus-visible identifier remains `sudharma-gpupow-v1` unless separately versioned.
- Do not change canonical Go/CUDA interoperability vectors to accommodate a backend.
- CPU implementation is consensus verifier/reference only, not supported production mining.
- GPU eligibility is derived from actual required allocation, not a hard-coded GPU model.
- No Seed-1/Seed-2 consensus activation before interoperability and deployment gates pass.
- Office RTX 2060 is the first physical test device, not a hard-coded target.
- Product compatibility target is vendor-neutral discrete GPUs with at least
  4 GiB dedicated VRAM: CUDA for NVIDIA and OpenCL 1.2+ for AMD/other vendors.
- A 4 GiB label does not bypass capability or usable-allocation preflight.
- Returned GPU solutions must be independently re-verified before submission.

---

### Task 1: Finish CUDA search and stale-work contract

**Files:**
- Create: `compatibility/cuda/gpupow_v1_search.cuh`
- Create: `compatibility/cuda/khushi_miner.cu`
- Modify: `compatibility/cuda/gpupow_v1.cu`
- Test: `compatibility/gpupowv1/cuda_search_contract_test.go`

**Interfaces:**
- Consumes: verified `final_digest_from_header(...)` and existing program/dataset primitives.
- Produces: `khushi_search_kernel`, `KhushiSearchResult`, explicit `nonce_start`, `nonce_count`, `work_generation`, `stale_generation` contract.

- [ ] Write/retain failing test requiring CUDA search kernel, atomic result selection, stale generation, explicit nonce range, benchmark branding and no CPU fallback.
- [ ] Run full CI and record RED failure caused by missing CUDA search files/contracts.
- [ ] Implement search contract with deterministic digest/target comparison and atomic first-result selection.
- [ ] Add host launcher that allocates device result/stale state, launches bounded nonce ranges and reports hashes attempted.
- [ ] Preserve refusal of `--mine` in non-CUDA builds.
- [ ] Run `go test ./... -count=1` and CUDA build checks available in CI.
- [ ] Commit GREEN and verify full Actions success.

### Task 2: Device capability and memory eligibility

**Files:**
- Create: `compatibility/gpupowv1/device_memory_test.go`
- Create: `compatibility/cuda/khushi_device.h`
- Modify: `compatibility/cuda/khushi_miner.cu`

**Interfaces:**
- Produces: `required_vram_bytes(cache_nodes, dag_nodes, runtime_reserve)`, explicit device rejection diagnostics and device enumeration metadata.

- [ ] Write failing tests proving eligibility is derived from allocation requirements and not from the text `RTX 2060` or a fixed `4GB` model rule.
- [ ] Verify RED.
- [ ] Implement overflow-safe required-memory calculation, CUDA device enumeration and allocation preflight.
- [ ] Freeze production epoch/cache sizing below the measured usable-memory
      ceiling of supported 4 GiB CUDA and OpenCL devices, including reserve.
- [ ] Add `--list-devices` and `--device N` parsing.
- [ ] Verify GREEN and full regression suite.
- [ ] Commit.

### Task 3: Benchmark mode and telemetry contract

**Files:**
- Create: `compatibility/gpupowv1/benchmark_contract_test.go`
- Modify: `compatibility/cuda/khushi_miner.cu`
- Create: `docs/khushi-miner.md`

**Interfaces:**
- Produces: bounded `--benchmark` mode that never submits, and stable machine-readable lines for backend/device/VRAM/hashrate/runtime plus optional temperature/power/utilization.

- [ ] Write failing CLI/source-contract tests for benchmark isolation and telemetry field names.
- [ ] Verify RED.
- [ ] Implement bounded benchmark execution and counters.
- [ ] Query optional NVIDIA telemetry through `nvidia-smi` only from host-side reporting when present; absence is non-fatal.
- [ ] Verify GREEN and full regression.
- [ ] Commit.

### Task 4: OpenCL backend contract

**Files:**
- Create: `compatibility/opencl/khushi_pow.cl`
- Create: `compatibility/opencl/khushi_miner_opencl.cpp`
- Create: `compatibility/gpupowv1/opencl_contract_test.go`

**Interfaces:**
- Consumes: same canonical domains, cache/DAG format, program seed, lane/mix/reduction and final-digest semantics.
- Produces: OpenCL device enumeration, memory preflight, search kernel, nonce-range/stale-generation contract and no-CPU-fallback host executable.

- [ ] Write failing source/vector contract tests requiring the exact algorithm/domain constants and shared search semantics.
- [ ] Verify RED.
- [ ] Port canonical 32-bit/KISS99/FNV/rotation/merge/dataset/final-digest primitives to OpenCL C without changing vectors.
- [ ] Implement OpenCL search kernel and host dispatch contract.
- [ ] Add vendor-neutral device selection and memory diagnostics.
- [ ] Run the unchanged canonical-vector and bounded-benchmark gates on at
      least one AMD OpenCL GPU with 4 GiB or more before claiming AMD support.
- [ ] Verify source/vector contract tests and full Go regression suite.
- [ ] Commit.

### Task 5: Windows GPU artifact CI

**Files:**
- Create: `.github/workflows/khushi-gpu-miner-windows.yml`
- Create: `scripts/windows/test-khushi-miner.ps1`
- Test: workflow YAML/source checks in `compatibility/gpupowv1/windows_artifact_contract_test.go`

**Interfaces:**
- Produces: revision-tagged Windows NVIDIA CUDA artifact and OpenCL-capable artifact where runner/toolkit availability permits, SHA-256 metadata and self-test/benchmark script.

- [ ] Write failing workflow contract tests for Windows build, artifact upload, commit SHA metadata and checksum generation.
- [ ] Verify RED.
- [ ] Implement Windows workflow using an explicitly installed/available CUDA toolkit path and CMake/nvcc or direct nvcc build.
- [ ] Package executable, README, revision and SHA256SUMS.
- [ ] Add PowerShell hardware test script that runs `nvidia-smi`, device listing, self-test and bounded benchmark without block submission.
- [ ] Verify workflow syntax/contracts and Linux regression CI.
- [ ] Commit and verify artifact-producing workflow result.

### Task 6: External mining API integration

**Files:**
- Create: `compatibility/gpupowv1/miner_work_contract_test.go`
- Modify/Create host miner networking source under `compatibility/miner/` as appropriate.
- Update: `docs/khushi-miner.md`

**Interfaces:**
- Consumes: `/v1/mining/work` and `/v1/mining/submit` only.
- Produces: work polling, immutable work-ID binding, generation increment on new work, GPU dispatch, stale cancellation and host-side solution re-verification before submit.

- [ ] Write failing tests for work-ID mutation rejection, generation rollover and no admin RPC dependency.
- [ ] Verify RED.
- [ ] Implement strict HTTP work client with bounded JSON/body sizes.
- [ ] Bind each GPU dispatch to work ID/height/header/target/reward address.
- [ ] Re-verify candidate nonce before submit and count accepted/rejected/stale results.
- [ ] Verify GREEN and full regression.
- [ ] Commit.

### Task 7: Hardware interoperability instructions and gate

**Files:**
- Update: `docs/khushi-miner.md`
- Update: `scripts/windows/test-khushi-miner.ps1`

**Interfaces:**
- Produces: one reproducible office-PC procedure whose output includes GPU model, driver, VRAM, self-test result, benchmark H/s, optional telemetry and artifact checksum.

- [ ] Document exact Windows prerequisites and commands.
- [ ] Ensure script performs benchmark only by default and requires an explicit staging endpoint flag for controlled solution submission.
- [ ] Verify documentation commands against produced CLI help text in CI where possible.
- [ ] Commit.

## Completion Gate

Software-side completion requires all repository CI and artifact workflows to pass and no CPU mining fallback to exist. Hardware interoperability remains pending until a real supported GPU executes the artifact, passes canonical vectors and returns a nonce independently accepted by the Go verifier. Seed-1/Seed-2 activation remains blocked until that evidence and the original deployment gates are satisfied.
