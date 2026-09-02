# Khushi Hardware Test v0.2.0 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build and publish an immutable, exact-revision Windows CUDA/OpenCL physical-GPU evidence package for Khushi Algorithm that runs canonical vectors, production-memory checks, bounded autotuning, a true full-duration selected-profile benchmark, and localhost independent staging verification.

**Architecture:** Preserve the existing CUDA/OpenCL miner binaries and Windows build workflows. Add only the missing benchmark semantics and OpenCL kernel-specific launch bound, then assemble exact-SHA successful backend artifacts into one combined v0.2.0 package with a one-click launcher and immutable release automation. Network mining remains gated; localhost staging is the only submission path in this package.

**Tech Stack:** Go contract/regression tests, C++17/CUDA 12.8, OpenCL 1.2, PowerShell, Windows batch, GitHub Actions, GitHub CLI.

**Spec:** `docs/test-mining/KHUSHI_HARDWARE_TEST_V0_2_0.md`

## Global Constraints

- Never modify `main` in this plan; work only on `feature/khushi-hardware-test-v0.2.0`.
- Preserve `MainnetLaunchAuthorized = false`.
- Preserve `MainnetMiningAuthorized = false`.
- Preserve `MainnetGenesisTimestamp = 0`.
- Preserve `PhysicalGPUMiningEvidenceComplete = false` until retained physical runs are reviewed.
- Preserve `PublicCommunitySecurityReviewComplete = false`.
- Default network remains `sudharma-testnet-1`.
- `--mine` remains gated.
- Local verifier binds only `127.0.0.1:28646`; no block creation, Seed-1/Seed-2 mutation, AWS mutation, key mutation, or live-testnet activation.
- NVIDIA and OpenCL package artifacts must record the exact same `source_revision=$GITHUB_SHA`.
- Release tag `khushi-hardware-test-v0.2.0` is create-once and must never be force-moved.
- Formal evidence command uses `-BenchmarkSeconds 60`.

---

### Task 1: Add failing contracts for physical benchmark semantics and OpenCL kernel safety

**Files:**
- Create: `compatibility/gpupowv1/hardware_test_v0_2_0_contract_test.go`
- Modify: `.github/workflows/gpu-pow-v1-ci.yml`

**Interfaces:**
- Consumes: CUDA `run_benchmark`, OpenCL `benchmark`, existing tuning helpers and Windows workflows.
- Produces: regression contracts that require short autotune sampling plus a separate full requested-duration final run and OpenCL `CL_KERNEL_WORK_GROUP_SIZE` enforcement.

- [ ] **Step 1: Write the failing Go source-contract tests**

Add tests that read `compatibility/cuda/khushi_miner.cu` and `compatibility/opencl/khushi_miner_opencl.cpp` and require these behavior markers:

```go
func TestCUDABenchmarkRunsSelectedProfileForFullRequestedDuration(t *testing.T) {
    // Require separate autotune candidate window and a final selected-profile deadline
    // based on std::chrono::seconds(seconds), plus final-benchmark output markers.
}

func TestOpenCLBenchmarkRunsSelectedProfileForFullRequestedDuration(t *testing.T) {
    // Same semantic requirement for OpenCL.
}

func TestOpenCLBenchmarkBoundsCandidatesByCompiledKernelLimit(t *testing.T) {
    // Require clGetKernelWorkGroupInfo, CL_KERNEL_WORK_GROUP_SIZE,
    // and candidate generation from the kernel-bounded local size.
}
```

The tests must fail on the current implementation because the final benchmark still reports the short winning autotune sample and OpenCL only uses the device-wide maximum.

- [ ] **Step 2: Run the focused tests and verify RED**

Run in CI/local Go environment:

```bash
go test ./compatibility/gpupowv1 -run 'BenchmarkRunsSelectedProfileForFullRequestedDuration|OpenCLBenchmarkBoundsCandidatesByCompiledKernelLimit' -count=1 -v
```

Expected: FAIL for missing separate final-run markers and missing `CL_KERNEL_WORK_GROUP_SIZE` query.

- [ ] **Step 3: Ensure the GPU-PoW CI workflow triggers on this feature branch**

Add `feature/khushi-hardware-test-v0.2.0` to `.github/workflows/gpu-pow-v1-ci.yml` push branches. Do not change consensus test commands.

- [ ] **Step 4: Commit the RED contracts**

Commit message:

```text
test(gpu): define hardware test v0.2.0 benchmark contracts
```

---

### Task 2: Make CUDA autotuning select, then benchmark the winning profile for the full requested duration

**Files:**
- Modify: `compatibility/cuda/khushi_miner.cu`
- Test: `compatibility/gpupowv1/hardware_test_v0_2_0_contract_test.go`

**Interfaces:**
- Consumes: `tuning::cuda_profile`, `tuning::candidates`, `tuning::work_items`, `khushi_search_kernel`.
- Produces: final CUDA benchmark line measured from a separate selected-profile run lasting at least `seconds` seconds.

- [ ] **Step 1: Keep autotune samples short and bounded**

Replace the old behavior that divides the requested benchmark duration among candidates with a fixed short candidate sample window. Use a named constant such as:

```cpp
constexpr unsigned long long kAutotuneCandidateMilliseconds = 1000ull;
```

Each candidate is measured for that bounded window only.

- [ ] **Step 2: Preserve selection output separately**

Keep `autotune-candidate` lines and `autotune-selected`, with `autotune-selected` describing the winning short sample.

- [ ] **Step 3: Reset device result buffers before the final run**

Reset `found_nonce` and `hashes_done`, set `nonce_start = 0`, and compute blocks/work items from the winning candidate.

- [ ] **Step 4: Run the selected CUDA profile for the requested duration**

Use a separate final timing window:

```cpp
const auto final_started = std::chrono::steady_clock::now();
const auto final_deadline = final_started + std::chrono::seconds(seconds);
```

Launch the winning geometry repeatedly until the deadline, then copy the final hash count and calculate final H/s.

- [ ] **Step 5: Make the final benchmark output unambiguous**

Emit a line containing:

```text
selected-profile-benchmark backend=cuda ... requested_seconds=N seconds=... hashes=... hashrate_hps=...
```

Then keep the compatibility line:

```text
Khushi Algorithm benchmark backend=cuda ...
```

Both lines must use the final selected-profile measurement, not the autotune sample.

- [ ] **Step 6: Run focused tests and full GPU-PoW CI**

Focused:

```bash
go test ./compatibility/gpupowv1 -run 'CUDABenchmarkRunsSelectedProfileForFullRequestedDuration|CUDABenchmarkUsesRuntimeAutotuneProfiles' -count=1 -v
```

Then require the branch GPU-PoW workflow to pass without changing activation defaults.

- [ ] **Step 7: Commit**

Commit message:

```text
feat(gpu): run full-duration CUDA selected-profile benchmark
```

---

### Task 3: Enforce OpenCL kernel-specific launch limits and full-duration selected-profile benchmark

**Files:**
- Modify: `compatibility/opencl/khushi_miner_opencl.cpp`
- Test: `compatibility/gpupowv1/hardware_test_v0_2_0_contract_test.go`

**Interfaces:**
- Consumes: compiled `khushi_search` kernel, `DeviceRef.max_work_group`, `tuning::opencl_profile`, `tuning::candidates`, `tuning::work_items`.
- Produces: kernel-bounded OpenCL candidate set and separate full-duration final selected-profile result.

- [ ] **Step 1: Query the compiled kernel limit before candidate generation**

After `clCreateKernel(rt.program, "khushi_search", ...)`, query:

```cpp
std::size_t kernel_max_work_group = 0;
check(clGetKernelWorkGroupInfo(
    kernel,
    chosen.device,
    CL_KERNEL_WORK_GROUP_SIZE,
    sizeof(kernel_max_work_group),
    &kernel_max_work_group,
    nullptr), "CL_KERNEL_WORK_GROUP_SIZE");
```

- [ ] **Step 2: Bound candidate local size by both device and kernel**

Compute the safe local limit as the minimum of the nonzero device maximum and nonzero kernel maximum. Fail closed if it resolves to zero. Generate `tuning::candidates` only after this limit is known.

- [ ] **Step 3: Use short bounded autotune samples**

Use the same 1000 ms per-candidate sampling semantics as CUDA.

- [ ] **Step 4: Reset OpenCL result buffers before the final run**

Reset hash count, found flag and nonce, then use the winning local/global geometry.

- [ ] **Step 5: Run the selected OpenCL profile for the requested duration**

Use a separate `final_started` / `final_deadline = final_started + std::chrono::seconds(seconds)` loop and read the final hash count afterward.

- [ ] **Step 6: Emit explicit final result**

Emit:

```text
selected-profile-benchmark backend=opencl ... requested_seconds=N seconds=... hashes=... hashrate_hps=...
```

and make `Khushi Algorithm benchmark backend=opencl ...` report the same final measurement.

- [ ] **Step 7: Run focused tests plus the Windows OpenCL build**

```bash
go test ./compatibility/gpupowv1 -run 'OpenCLBenchmarkRunsSelectedProfileForFullRequestedDuration|OpenCLBenchmarkBoundsCandidatesByCompiledKernelLimit|OpenCLBenchmarkUsesRuntimeAutotuneProfiles' -count=1 -v
```

Require `Khushi Windows OpenCL` to compile successfully at the exact branch head.

- [ ] **Step 8: Commit**

Commit message:

```text
feat(gpu): harden OpenCL autotune and final benchmark
```

---

### Task 4: Add one-click physical runner and exact-revision immutable package publisher

**Files:**
- Create: `scripts/windows/Run-GPU-Test.bat`
- Create: `docs/test-mining/KHUSHI_HARDWARE_TEST_README.txt`
- Create: `.github/workflows/publish-khushi-hardware-test-v0.2.0.yml`
- Create: `compatibility/gpupowv1/hardware_test_release_contract_test.go`
- Modify: `.github/workflows/khushi-windows-cuda.yml`
- Modify: `.github/workflows/khushi-windows-opencl.yml`

**Interfaces:**
- Consumes: exact-SHA artifacts `khushi-miner-nvidia-windows` and `khushi-miner-opencl-windows` from successful backend workflows.
- Produces: `khushi-hardware-test-v0.2.0-windows.zip`, release checksum/source-revision files and immutable prerelease tag `khushi-hardware-test-v0.2.0`.

- [ ] **Step 1: Write failing release/package contracts**

Require:

- both Windows workflows trigger on `feature/khushi-hardware-test-v0.2.0`;
- publisher resolves successful CUDA/OpenCL runs where `.headSha == $GITHUB_SHA`;
- publisher verifies `source_revision=$GITHUB_SHA` in both backend metadata files;
- combined package contains `nvidia/`, `opencl/`, `Run-GPU-Test.bat`, `README.txt`, `PACKAGE-METADATA.txt`, and `SHA256MANIFEST.txt`;
- tag is exactly `khushi-hardware-test-v0.2.0`;
- publisher checks branch head equals `GITHUB_SHA` before publication;
- publisher contains no `force=true`, no forced tag update and no `--clobber` release upload;
- existing tag at a different SHA fails closed.

Run:

```bash
go test ./compatibility/gpupowv1 -run 'HardwareTestRelease|HardwareTestPackage' -count=1 -v
```

Expected: RED because the new workflow and launcher do not exist.

- [ ] **Step 2: Add the feature branch to backend Windows workflow triggers**

Add `feature/khushi-hardware-test-v0.2.0` to CUDA and OpenCL workflow push branches. Add the new launcher/readme/release-contract paths to relevant path filters so exact-head builds rerun when package behavior changes.

- [ ] **Step 3: Implement `Run-GPU-Test.bat`**

The batch file must:

- default to device `0` when the user presses Enter;
- offer NVIDIA CUDA when `nvidia/khushi-miner-nvidia.exe` exists;
- offer OpenCL when `opencl/khushi-miner-opencl.exe` exists;
- invoke the chosen backend's `run-local-staging-gate.ps1` with explicit verifier/miner paths, selected device, and `-BenchmarkSeconds 60`;
- never invoke `--mine` or a public endpoint.

- [ ] **Step 4: Add a concise package README**

Document that the user should extract the ZIP, run `Run-GPU-Test.bat`, select backend/device, and retain the generated `khushi-staging-evidence-*` directory. State that success requires `local-staging-gate=accepted` and does not activate network/mainnet mining.

- [ ] **Step 5: Implement exact-revision publisher workflow**

Create `.github/workflows/publish-khushi-hardware-test-v0.2.0.yml` with `contents: write` and `actions: read`. It must run only on the dedicated branch and only publish when the head commit message is exactly:

```text
release(gpu): publish Khushi hardware test v0.2.0
```

Resolve exact-SHA successful backend runs with a bounded poll, download both artifacts, verify required files and source revisions, compose one combined package, write SHA256 manifest and package metadata, and verify the dedicated branch head still equals `GITHUB_SHA`.

- [ ] **Step 6: Make tag publication immutable**

If `khushi-hardware-test-v0.2.0` does not exist, create the prerelease targeting `GITHUB_SHA` and upload assets without `--clobber`. If it exists, resolve its SHA and fail if it differs from `GITHUB_SHA`; never patch or force-move the tag.

- [ ] **Step 7: Run package/release contracts and full GPU-PoW CI**

```bash
go test ./compatibility/gpupowv1 -run 'HardwareTestRelease|HardwareTestPackage|WindowsMinerPackagesContainSameRevisionLocalStagingVerifier' -count=1 -v
go test ./compatibility/gpupowv1 -count=1
```

Then require exact-head GPU-PoW CI, CUDA Windows build and OpenCL Windows build success.

- [ ] **Step 8: Commit**

Commit message before publication:

```text
feat(gpu): package Khushi hardware test v0.2.0
```

---

### Task 5: Publish the exact verified package and independently inspect the artifact

**Files:**
- No consensus/runtime source changes after the release marker commit.
- Release commit changes only a harmless release-triggered file if needed to create the exact release marker commit without altering tested behavior.

**Interfaces:**
- Consumes: green exact-head GPU-PoW, CUDA and OpenCL workflow runs.
- Produces: immutable public prerelease assets and a user-downloadable combined ZIP.

- [ ] **Step 1: Re-check `main` and all safety flags before release marker**

Verify current `main` and confirm no change in mainnet launch/mining/genesis or physical/public-review evidence flags.

- [ ] **Step 2: Create the release marker commit without changing tested behavior**

Use exact commit message:

```text
release(gpu): publish Khushi hardware test v0.2.0
```

A documentation-only release record may be used for the marker commit. This commit itself becomes the artifact source revision, so all exact-head Windows workflows must build that SHA.

- [ ] **Step 3: Require all exact-SHA workflows green**

Do not claim readiness until:

- GPU-PoW v1 CI success;
- Khushi NVIDIA Windows CUDA success;
- Khushi Windows OpenCL success;
- immutable publisher success.

- [ ] **Step 4: Download the combined workflow/release artifact and inspect it**

Verify ZIP structure contains both backends, one-click launcher, README, exact source revision and SHA256 manifest. Verify internal backend metadata source revisions equal the release commit SHA.

- [ ] **Step 5: Recompute package SHA256 outside the publishing workflow**

Record the independently recomputed ZIP SHA256 and compare it with `RELEASE-SHA256SUMS.txt`.

- [ ] **Step 6: Hand the combined ZIP to the user with the formal command path**

Tell the user to extract and run `Run-GPU-Test.bat`; formal success evidence is the generated evidence folder ending with `local-staging-gate=accepted`.

- [ ] **Step 7: Do not close physical evidence gate yet**

The software package being green does not set `PhysicalGPUMiningEvidenceComplete=true`. That remains false until the user's real RTX run and a real AMD/non-NVIDIA OpenCL >=4 GiB run are retained and reviewed.

---

## Self-review

- Spec coverage: full-duration final benchmark, kernel-specific OpenCL limit, CUDA/OpenCL package, one-click runner, exact-SHA provenance, immutable tag, localhost-only verifier, evidence manifest and safety invariants are each mapped to tasks above.
- Placeholder scan: no TBD/TODO/deferred implementation steps remain.
- Type/interface consistency: both miners retain `--benchmark N`; final benchmark semantics are identical across backends; release consumes the existing backend artifact names and metadata key `source_revision`.
