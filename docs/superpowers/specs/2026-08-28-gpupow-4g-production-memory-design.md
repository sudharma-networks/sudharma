# GPU-PoW v1 4 GiB Production Memory Design

## Status

Pre-consensus design for Stage B. Nothing in this document activates GPU-PoW,
changes Seed-1/Seed-2, or changes the frozen interoperability vectors.

## Goal

Make Khushi Algorithm mining practical on vendor-neutral discrete GPUs with
4 GiB or more dedicated VRAM while retaining light CPU verification, identical
CUDA/OpenCL results, and a meaningful GPU memory-hard dataset.

## Current gap

The current eight-node, 512-byte cache is an interoperability fixture. CUDA and
OpenCL derive dataset items from it on demand. This proves cross-implementation
digest agreement but does not represent a production full-DAG allocation or a
credible 4 GiB hardware-compatibility test.

## Considered approaches

### A. Fixed 2 GiB dataset for GPU-PoW v1 — recommended

Freeze the v1 dataset size so supported 4 GiB cards remain usable for the life
of v1. A later algorithm version may deliberately revise the memory schedule.
This provides predictable hardware support and prevents ordinary epoch growth
from silently excluding the user’s target devices.

### B. Growing dataset capped below 4 GiB

This follows classic Ethash/ProgPoW behavior more closely, but driver-reserved
memory and operating-system variation make the eventual cap a compatibility
cliff. It also makes the date at which a nominal 4 GiB card fails dependent on
implementation details. This is rejected for v1.

### C. Light-cache-only mining

This is easiest to run but does not provide the intended full-DAG GPU memory
pressure. It remains useful for deterministic vectors and CPU verification,
not supported production mining.

## Proposed v1 allocation

| Allocation | Size | Purpose |
|---|---:|---|
| Full epoch dataset | 2 GiB | GPU-resident memory-hard search data |
| Light cache | 16 MiB | Deterministic dataset generation and CPU verification |
| Runtime reserve | 256 MiB | Results, headers, queues, driver/runtime working space |
| Dataset chunk | 256 MiB maximum | Portable allocation unit for CUDA/OpenCL |

The nominal required allocation is 2.265625 GiB before backend-specific
overhead. Admission still uses measured usable memory and actual allocation
success. Total advertised VRAM alone is not sufficient.

The dataset contains 33,554,432 64-byte items. The light cache contains 262,144
64-byte nodes. These values are proposed inputs to a new versioned production
memory contract; they must not replace the existing fixture or enter active
consensus until the gates below pass.

## Backend contract

CUDA and OpenCL consume the same epoch seed and generate byte-identical dataset
items. Logical dataset indexing is independent of physical allocation layout.
The 2 GiB dataset is exposed as eight 256 MiB chunks so OpenCL devices are not
forced to accept one multi-gigabyte memory object. CUDA may use one contiguous
allocation internally only if its output remains identical.

Device admission requires:

1. a discrete GPU backend, never a CPU fallback;
2. CUDA or an OpenCL 1.2+ GPU runtime;
3. reported dedicated VRAM of at least 4 GiB;
4. sufficient usable memory after driver reservations; and
5. successful allocation and initialization of every required buffer.

Failure at any step is explicit and does not change consensus behavior.

## Verification and activation gates

1. Add pure Go sizing/index/chunk vectors and overflow tests.
2. Add CUDA and OpenCL dataset-generation vectors at chunk boundaries.
3. Prove the same digest for items 0, last-in-chunk, first-in-next-chunk, and
   final dataset item.
4. Run allocation, vector, benchmark, and controlled staging submission on the
   office RTX 2060.
5. Run the identical gate on at least one AMD/OpenCL GPU with 4 GiB or more.
6. Have the returned nonces independently accepted by the Go verifier.
7. Review mixed-version behavior, rollback, and denial-of-service costs.
8. Only then create a separate activation proposal for Seed-1/Seed-2.

## Evidence basis

- The ProgPoW specification uses separate cache and dataset schedules and a
  full dataset much larger than its light cache:
  https://github.com/MariusVanDerWijden/progpow-wiki/blob/master/ProgPoW.md
- Khronos defines `CL_DEVICE_GLOBAL_MEM_SIZE` and
  `CL_DEVICE_MAX_MEM_ALLOC_SIZE`; portable allocation design must respect both:
  https://registry.khronos.org/OpenCL/specs/unified/html/OpenCL_API.html
- NVIDIA documents `cudaMemGetInfo` for available and total device memory:
  https://docs.nvidia.com/cuda/cuda-runtime-api/group__CUDART__MEMORY.html

The exact 2 GiB/16 MiB/256 MiB values are a Sudharma design inference chosen to
leave substantial headroom on the 4 GiB product target. They are not copied
from another chain and remain provisional until real CUDA and OpenCL evidence
passes.
