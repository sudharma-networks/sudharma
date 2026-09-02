# Khushi GPU Autotune Profiles Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add model-neutral NVIDIA/AMD GPU launch profiling and bounded self-benchmark autotuning so Khushi mining no longer relies on the RTX 2060 launch geometry.

**Architecture:** Keep GPU model names and external mining benchmark data advisory only. Runtime facts (CUDA compute capability/SM count/max threads or OpenCL vendor/compute units/max work-group size) generate a small safe candidate set; the native Khushi benchmark measures those candidates and chooses the fastest stable geometry. Consensus inputs, hashes, targets, nonces, memory policy, CPU-fallback prohibition, and activation defaults remain unchanged.

**Tech Stack:** C++17 shared profile helper, CUDA runtime, OpenCL 1.2, Go contract tests, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-08-28-khushi-multivendor-gpu-miner-design.md`

## Global Constraints

- NVIDIA uses CUDA; AMD and other compatible devices use OpenCL.
- CPU mining fallback must never activate silently.
- GPU brand/model is never a consensus input.
- Public compatibility target remains discrete GPUs with at least 4 GiB dedicated VRAM and runtime allocation preflight.
- External mining-platform hashrates are tuning hints only; never label them as measured Khushi H/s.
- Physical RTX 2060 and AMD/non-NVIDIA evidence gates remain separate and unchanged.
- GPU-PoW network activation remains disabled.

---

### Task 1: Shared launch-profile policy

**Files:**
- Create: `compatibility/gpu/gpu_tuning_profile.h`
- Create: `compatibility/gpu/gpu_tuning_profile_test.cpp`
- Create: `compatibility/gpupowv1/gpu_tuning_profile_contract_test.go`

**Interfaces:**
- Consumes: runtime vendor/family facts and maximum local/work-group size.
- Produces: `sudharma::gpupowv1::tuning::Profile`, `Candidate`, `cuda_profile(...)`, `opencl_profile(...)`, and `candidates(...)`.

- [ ] **Step 1: Write the failing C++ test**

```cpp
#include "gpu_tuning_profile.h"
#include <cassert>
using namespace sudharma::gpupowv1::tuning;
int main() {
    assert(cuda_profile(7, 5).family == Family::NvidiaTuring);
    assert(cuda_profile(8, 6).family == Family::NvidiaAmpere);
    assert(cuda_profile(8, 9).family == Family::NvidiaAda);
    assert(cuda_profile(12, 0).family == Family::NvidiaBlackwell);
    assert(opencl_profile("Advanced Micro Devices, Inc.").family == Family::AmdGeneric);
    auto safe = candidates(Profile{Family::Generic, 32}, 64);
    assert(!safe.empty());
    for (auto c : safe) assert(c.local_size <= 64);
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `g++ -std=c++17 -I compatibility/gpu compatibility/gpu/gpu_tuning_profile_test.cpp -o /tmp/gpu-profile-test`
Expected: FAIL because `gpu_tuning_profile.h` does not exist.

- [ ] **Step 3: Implement the minimal header**

Implement family classification plus deterministic, duplicate-free safe candidates. NVIDIA families use 32-thread warp-aware starting points; AMD starts from 64-thread wave-aware candidates; generic fallback remains conservative and every candidate is clamped to the runtime maximum.

- [ ] **Step 4: Run test to verify it passes**

Run: `g++ -std=c++17 -I compatibility/gpu compatibility/gpu/gpu_tuning_profile_test.cpp -o /tmp/gpu-profile-test && /tmp/gpu-profile-test`
Expected: PASS.

- [ ] **Step 5: Add a Go source/compile contract**

The Go contract test must compile and execute the C++ policy test when `g++` is available and assert the CUDA/OpenCL miners include the shared header after Tasks 2-3.

### Task 2: CUDA benchmark autotune

**Files:**
- Modify: `compatibility/cuda/khushi_miner.cu`
- Modify: `compatibility/gpupowv1/cuda_benchmark_cli_contract_test.go`

**Interfaces:**
- Consumes: `cuda_profile(prop.major, prop.minor)` and runtime `multiProcessorCount` / `maxThreadsPerBlock`.
- Produces: candidate benchmark lines and one `autotune-selected` line containing family, local size, work items, and measured Khushi H/s.

- [ ] **Step 1: Extend the Go contract test first**

Require source tokens for `gpu_tuning_profile.h`, `cuda_profile`, `autotune-selected`, `maxThreadsPerBlock`, and non-single-block grid sizing.

- [ ] **Step 2: Run the contract test and confirm RED**

Run: `go test ./compatibility/gpupowv1 -run 'CUDABenchmark|GPUTuningProfile' -count=1`
Expected: FAIL because CUDA is still fixed at 32 threads / 32 nonces.

- [ ] **Step 3: Implement CUDA candidate benchmarking**

For each safe candidate, set `threads = local_size`, derive bounded `work_items` from SM count and candidate groups-per-unit, launch enough blocks to cover that work, measure actual hashes/time, keep the best valid H/s, and print the chosen geometry. Preserve target, digest, nonce and independent-verification semantics.

- [ ] **Step 4: Re-run contract tests**

Expected: PASS.

### Task 3: OpenCL benchmark autotune

**Files:**
- Modify: `compatibility/opencl/khushi_miner_opencl.cpp`
- Create: `compatibility/gpupowv1/opencl_benchmark_autotune_contract_test.go`

**Interfaces:**
- Consumes: `CL_DEVICE_VENDOR`, `CL_DEVICE_MAX_COMPUTE_UNITS`, `CL_DEVICE_MAX_WORK_GROUP_SIZE` and `opencl_profile(...)`.
- Produces: candidate benchmark lines and one `autotune-selected` line.

- [ ] **Step 1: Write RED source contract**

Require runtime vendor/compute-unit/max-work-group queries, shared profile inclusion, explicit local/global work sizes larger than one when permitted, and `autotune-selected` output.

- [ ] **Step 2: Run and confirm RED**

Run: `go test ./compatibility/gpupowv1 -run 'OpenCLBenchmarkAutotune' -count=1`
Expected: FAIL because the current benchmark launches exactly one OpenCL work item.

- [ ] **Step 3: Implement OpenCL autotune**

Extend `DeviceRef` with vendor, compute units, and maximum work-group size; benchmark each clamped candidate using `global = local * compute_units * groups_per_unit`; select the fastest stable result. Do not infer eligibility from the model name.

- [ ] **Step 4: Re-run tests**

Expected: PASS.

### Task 4: Provenance and CI guard

**Files:**
- Create: `docs/khushi-gpu-autotune.md`
- Modify: `.github/workflows/gpu-pow-v1-ci.yml`

**Interfaces:**
- Documents official NVIDIA compute-capability and AMD architecture/runtime references, plus Kryptex/Hashrate.no-style mining data as advisory background only.

- [ ] **Step 1: Document evidence classes**

Define `external-profile`, `locally-benchmarked`, and `physically-verified`; explicitly state that only the last class satisfies hardware evidence gates.

- [ ] **Step 2: Add the feature branch to GPU-PoW CI push coverage**

Add `feature/gpu-autotune-profiles` to the workflow branch list and run the focused contracts plus full Go regression.

- [ ] **Step 3: Verify activation safety**

Run: `go test ./pow -run TestGPUV1NetworkActivationDefaultsRemainDisabled -count=1 -v`
Expected: PASS.

- [ ] **Step 4: Final verification**

Run: `go test ./... -count=1` and the shared C++ policy test.
Expected: PASS; physical GPU evidence flags remain unchanged.
