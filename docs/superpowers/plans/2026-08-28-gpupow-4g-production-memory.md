# GPU-PoW v1 4 GiB Production Memory Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build and verify a pre-activation 2 GiB chunked epoch dataset contract that can be mined identically by 4 GiB-class CUDA and OpenCL GPUs.

**Architecture:** Add a staging-only Go memory policy and logical chunk mapper first, then bind CUDA and OpenCL dataset generation to the same vectors. Keep the existing eight-node fixtures unchanged. Do not connect the new parameters to active block validation or Seed-1/Seed-2 until physical evidence and a separate activation proposal pass.

**Tech Stack:** Go 1.26 tests/reference code, CUDA C++, OpenCL 1.2 C/C++, GitHub Actions, Windows PowerShell.

**Spec:** `docs/superpowers/specs/2026-08-28-gpupow-4g-production-memory-design.md`

## Global Constraints

- Full epoch dataset: exactly 2 GiB (`2 * 1024 * 1024 * 1024` bytes).
- Light cache: exactly 16 MiB (`16 * 1024 * 1024` bytes).
- Runtime reserve: exactly 256 MiB (`256 * 1024 * 1024` bytes).
- Dataset chunk: exactly 256 MiB, producing eight chunks.
- Dataset item and cache node size: exactly 64 bytes.
- Minimum product target: discrete GPU with at least 4 GiB dedicated VRAM.
- CUDA and OpenCL must produce byte-identical items and final digests.
- Existing interoperability vectors and eight-node fixture remain unchanged.
- CPU is a reference/verifier only and never a production mining fallback.
- No consensus activation, seed deployment, mainnet, monetary, wallet, or IAM change.

---

### Task 1: Staging Memory Policy

**Files:**
- Create: `compatibility/gpupowv1/memory_policy.go`
- Create: `compatibility/gpupowv1/memory_policy_test.go`

**Interfaces:**
- Produces: `ProductionMemoryPolicy`, `GPUV1ProductionMemory`, `RequiredVRAMBytes() (uint64, bool)`, and `EligibleDedicatedVRAM(total, available uint64) bool`.

- [ ] **Step 1: Write the failing constants and arithmetic test**

```go
func TestProductionMemoryPolicyFitsFourGiBTarget(t *testing.T) {
    p := GPUV1ProductionMemory
    if p.DatasetBytes != 2<<30 || p.CacheBytes != 16<<20 ||
        p.RuntimeReserveBytes != 256<<20 || p.ChunkBytes != 256<<20 ||
        p.ItemBytes != 64 {
        t.Fatalf("unexpected policy: %+v", p)
    }
    required, ok := p.RequiredVRAMBytes()
    if !ok || required != 2432696320 {
        t.Fatalf("required=%d ok=%v", required, ok)
    }
    if !p.EligibleDedicatedVRAM(4<<30, required) {
        t.Fatal("conforming 4 GiB device rejected")
    }
    if p.EligibleDedicatedVRAM((4<<30)-1, required) {
        t.Fatal("sub-4 GiB device accepted")
    }
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./compatibility/gpupowv1 -run TestProductionMemoryPolicyFitsFourGiBTarget -count=1`

Expected: FAIL because `GPUV1ProductionMemory` does not exist.

- [ ] **Step 3: Implement the immutable staging policy**

```go
type ProductionMemoryPolicy struct {
    DatasetBytes, CacheBytes, RuntimeReserveBytes uint64
    ChunkBytes, ItemBytes, MinimumDedicatedVRAMBytes uint64
}

var GPUV1ProductionMemory = ProductionMemoryPolicy{
    DatasetBytes: 2 << 30, CacheBytes: 16 << 20,
    RuntimeReserveBytes: 256 << 20, ChunkBytes: 256 << 20,
    ItemBytes: 64, MinimumDedicatedVRAMBytes: 4 << 30,
}

func (p ProductionMemoryPolicy) RequiredVRAMBytes() (uint64, bool) {
    if p.DatasetBytes > ^uint64(0)-p.CacheBytes { return 0, false }
    n := p.DatasetBytes + p.CacheBytes
    if n > ^uint64(0)-p.RuntimeReserveBytes { return 0, false }
    return n + p.RuntimeReserveBytes, true
}

func (p ProductionMemoryPolicy) EligibleDedicatedVRAM(total, available uint64) bool {
    required, ok := p.RequiredVRAMBytes()
    return ok && total >= p.MinimumDedicatedVRAMBytes && available >= required
}
```

- [ ] **Step 4: Add overflow and invalid-layout cases**

Use literal policies proving addition overflow, zero item size, and non-divisible dataset/chunk layouts are rejected by a new `Validate() error` method.

- [ ] **Step 5: Verify GREEN**

Run: `go test ./compatibility/gpupowv1 -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add compatibility/gpupowv1/memory_policy.go compatibility/gpupowv1/memory_policy_test.go
git commit -m "feat(gpu-pow): define staged 4 GiB memory policy"
```

### Task 2: Logical Dataset Chunk Mapping

**Files:**
- Create: `compatibility/gpupowv1/dataset_layout.go`
- Create: `compatibility/gpupowv1/dataset_layout_test.go`

**Interfaces:**
- Consumes: `ProductionMemoryPolicy`.
- Produces: `DatasetLocation{Chunk uint32, Offset uint64}` and `DatasetItemLocation(index uint64) (DatasetLocation, error)`.

- [ ] **Step 1: Write failing boundary-vector tests**

```go
func TestDatasetItemLocationChunkBoundaries(t *testing.T) {
    cases := []struct{ index uint64; chunk uint32; offset uint64 }{
        {0, 0, 0},
        {4194303, 0, 268435392},
        {4194304, 1, 0},
        {33554431, 7, 268435392},
    }
    for _, tc := range cases {
        got, err := GPUV1ProductionMemory.DatasetItemLocation(tc.index)
        if err != nil || got.Chunk != tc.chunk || got.Offset != tc.offset {
            t.Fatalf("index=%d got=%+v err=%v", tc.index, got, err)
        }
    }
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./compatibility/gpupowv1 -run TestDatasetItemLocationChunkBoundaries -count=1`

Expected: FAIL because `DatasetItemLocation` does not exist.

- [ ] **Step 3: Implement checked mapping**

Compute `byteOffset := index * ItemBytes` only after checking multiplication overflow and `index < DatasetBytes/ItemBytes`. Return `Chunk = byteOffset/ChunkBytes` and `Offset = byteOffset%ChunkBytes`.

- [ ] **Step 4: Add out-of-range and overflow tests**

Assert errors for index `33554432`, `^uint64(0)`, zero item size, and a policy whose chunk size is not a multiple of item size.

- [ ] **Step 5: Verify GREEN and commit**

Run: `go test ./compatibility/gpupowv1 -count=1`

```bash
git add compatibility/gpupowv1/dataset_layout.go compatibility/gpupowv1/dataset_layout_test.go
git commit -m "feat(gpu-pow): map production dataset chunks"
```

### Task 3: Canonical Production Dataset Boundary Vectors

**Files:**
- Create: `docs/gpu-pow-v1-production-memory-vectors.json`
- Create: `compatibility/gpupowv1/production_memory_vectors_test.go`
- Modify: `compatibility/gpupowv1/independent.go`

**Interfaces:**
- Consumes: epoch seed derivation, light-cache generation, and `DatasetItemLocation`.
- Produces: fixed epoch-0 vectors for item indices `0`, `4194303`, `4194304`, and `33554431`.

- [ ] **Step 1: Add a failing fixture-consumer test**

The test must parse a schema containing `algorithm`, `dataset_bytes`, `cache_bytes`, `chunk_bytes`, `item_bytes`, `epoch`, and four literal item hex digests. It must reject missing, duplicate, or out-of-range indices.

- [ ] **Step 2: Verify RED**

Run: `go test ./compatibility/gpupowv1 -run ProductionMemoryVectors -count=1`

Expected: FAIL because the fixture is absent.

- [ ] **Step 3: Extend the independent verifier without changing old vectors**

Add a streaming cache builder for exactly 262,144 nodes and a dataset-item function that accepts a `[]byte` cache. Do not modify `Digest(...)` or `docs/gpu-pow-v1-interoperability-vectors.json`.

- [ ] **Step 4: Generate and freeze literal vectors**

Use a one-time generator under a temporary directory, independently inspect the JSON fields and counts, then save only the resulting JSON fixture. Remove the generator before commit.

- [ ] **Step 5: Verify old and new vector suites**

Run: `go test ./compatibility/gpupowv1 ./pow -count=1`

Expected: PASS, including unchanged legacy vector tests.

- [ ] **Step 6: Commit**

```bash
git add docs/gpu-pow-v1-production-memory-vectors.json compatibility/gpupowv1/production_memory_vectors_test.go compatibility/gpupowv1/independent.go
git commit -m "test(gpu-pow): freeze production memory boundary vectors"
```

### Task 4: CUDA Chunked Dataset Gate

**Files:**
- Create: `compatibility/cuda/gpupow_v1_chunks.cuh`
- Modify: `compatibility/cuda/khushi_miner.cu`
- Create: `compatibility/gpupowv1/cuda_chunk_contract_test.go`

**Interfaces:**
- Consumes: Task 3 fixture and canonical CUDA dataset-item primitive.
- Produces: eight checked 256 MiB allocations and logical-item lookup independent of physical layout.

- [ ] **Step 1: Write a failing portable CUDA-header harness**

Compile the header with `g++` in host-contract mode and assert the four literal `(chunk, offset)` mappings from Task 2. The same test must require checked addition/multiplication and reject chunk `8`.

- [ ] **Step 2: Verify RED**

Run: `go test ./compatibility/gpupowv1 -run CUDAChunk -count=1`

Expected: FAIL because `gpupow_v1_chunks.cuh` is absent.

- [ ] **Step 3: Implement the layout header and CUDA allocation lifecycle**

Allocate exactly eight chunks after `cudaMemGetInfo` preflight. On partial failure, free every previously allocated chunk and return a non-zero error. Keep benchmark and staging fixture modes on their existing small cache until the production-vector gate explicitly selects the new mode.

- [ ] **Step 4: Add CUDA boundary-item workflow checks**

Extend `.github/workflows/khushi-windows-cuda.yml` to compile a vector-only executable that generates the four Task 3 items. Compare its output byte-for-byte with the JSON fixture; no mining submission is allowed.

- [ ] **Step 5: Verify and commit**

Run: `go test ./compatibility/gpupowv1 ./pow -count=1`

```bash
git add compatibility/cuda/gpupow_v1_chunks.cuh compatibility/cuda/khushi_miner.cu compatibility/gpupowv1/cuda_chunk_contract_test.go .github/workflows/khushi-windows-cuda.yml
git commit -m "feat(gpu-pow): gate chunked CUDA dataset"
```

### Task 5: OpenCL Chunked Dataset Gate

**Files:**
- Modify: `compatibility/opencl/khushi_miner_opencl.cpp`
- Modify: `compatibility/opencl/khushi_pow.cl`
- Create: `compatibility/gpupowv1/opencl_chunk_contract_test.go`
- Modify: `.github/workflows/gpu-pow-v1-opencl-ci.yml`
- Modify: `.github/workflows/khushi-windows-opencl.yml`

**Interfaces:**
- Consumes: Tasks 2–3 layout and vectors.
- Produces: eight OpenCL memory objects, max-allocation diagnostics, and byte-identical boundary items.

- [ ] **Step 1: Write failing host and kernel boundary tests**

Require device queries for `CL_DEVICE_GLOBAL_MEM_SIZE`, `CL_DEVICE_MAX_MEM_ALLOC_SIZE`, and `CL_DEVICE_HOST_UNIFIED_MEMORY`. Reject CPU devices and unified-memory devices for the dedicated-GPU production gate. Assert that every 256 MiB chunk is within the device max-allocation value.

- [ ] **Step 2: Verify RED**

Run: `go test ./compatibility/gpupowv1 -run OpenCLChunk -count=1`

Expected: FAIL because max-allocation/unified-memory admission is absent.

- [ ] **Step 3: Implement checked OpenCL allocation and cleanup**

Create eight `cl_mem` objects, release all successfully created objects on any later failure, and pass chunk handles plus logical index to the vector kernel. Do not fall back to a CPU OpenCL device.

- [ ] **Step 4: Compare all boundary vectors in Linux and Windows workflows**

Both workflows must emit the four literal item digests and fail on any mismatch. Artifact metadata records backend, source SHA, dataset/cache/chunk sizes, and OpenCL platform/device strings.

- [ ] **Step 5: Verify and commit**

Run: `go test ./compatibility/gpupowv1 ./pow -count=1`

```bash
git add compatibility/opencl/khushi_miner_opencl.cpp compatibility/opencl/khushi_pow.cl compatibility/gpupowv1/opencl_chunk_contract_test.go .github/workflows/gpu-pow-v1-opencl-ci.yml .github/workflows/khushi-windows-opencl.yml
git commit -m "feat(gpu-pow): gate chunked OpenCL dataset"
```

### Task 6: Physical 4 GiB Evidence Packages

**Files:**
- Modify: `scripts/windows/test-khushi-miner.ps1`
- Create: `scripts/windows/test-khushi-opencl.ps1`
- Modify: `docs/khushi-rtx2060-mining-test.md`
- Create: `docs/khushi-amd-opencl-mining-test.md`
- Modify: `compatibility/gpupowv1/hardware_test_package_contract_test.go`

**Interfaces:**
- Consumes: revision-matched CUDA/OpenCL artifacts and Task 3 vectors.
- Produces: tamper-evident CUDA and OpenCL evidence archives with no default network submission.

- [ ] **Step 1: Write failing packaging tests**

Require both scripts to record total/available/max-allocation memory, all eight allocation results, four boundary-vector results, bounded benchmark H/s, artifact/source SHA-256, driver/runtime, and `network-submission=not-requested` by default.

- [ ] **Step 2: Verify RED**

Run: `go test ./compatibility/gpupowv1 -run HardwareTestPackage -count=1`

Expected: FAIL because the OpenCL evidence script and production-memory fields are absent.

- [ ] **Step 3: Implement evidence collection**

Make CUDA and OpenCL scripts fail closed on checksum, allocation, vector, or benchmark failure. Controlled staging submission remains a separate explicit flag and accepts only a localhost or approved staging endpoint.

- [ ] **Step 4: Run software verification**

Run:

```bash
go test ./compatibility/gpupowv1 ./pow ./rpc -count=1
go test -race ./compatibility/gpupowv1 ./rpc -count=1
go vet ./...
go test ./... -count=1
```

Expected: all commands exit 0.

- [ ] **Step 5: Commit and publish draft-only artifacts**

```bash
git add scripts/windows/test-khushi-miner.ps1 scripts/windows/test-khushi-opencl.ps1 docs/khushi-rtx2060-mining-test.md docs/khushi-amd-opencl-mining-test.md compatibility/gpupowv1/hardware_test_package_contract_test.go
git commit -m "test(gpu-pow): package 4 GiB hardware evidence gates"
```

Physical CUDA and AMD/OpenCL executions remain pending until the corresponding
hardware is available. Their absence must keep consensus activation blocked.
