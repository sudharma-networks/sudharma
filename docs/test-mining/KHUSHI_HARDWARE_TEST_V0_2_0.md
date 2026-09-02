# Khushi Hardware Test v0.2.0

Status: approved design for implementation on `feature/khushi-hardware-test-v0.2.0`.

## Goal

Produce a self-contained Windows physical-GPU evidence package for `sudharma-gpupow-v1` that can be run on one selected CUDA or OpenCL GPU without enabling network mining, creating a block, changing consensus activation, or touching Seed-1/Seed-2 or AWS resources.

## Package contents

The combined Windows package must contain:

- `nvidia/khushi-miner-nvidia.exe`
- `nvidia/khushi-production-vectors-nvidia.exe`
- the CUDA runtime DLL required by the NVIDIA executable
- `opencl/khushi-miner-opencl.exe`
- `opencl/khushi-production-vectors-opencl.exe`
- `opencl/khushi_pow.cl`
- `opencl/gpupow_v1_production_vectors.cl`
- same-revision `sudharma-gpupow-staging.exe` in each backend directory or one package-level verified copy
- frozen production-vector JSON
- `test-khushi-miner.ps1`
- `run-local-staging-gate.ps1`
- `Run-GPU-Test.bat`
- exact source-revision metadata and SHA256 manifests
- a short `README.txt` with NVIDIA/OpenCL selection instructions

## Required hardware flow

For the selected GPU, the test must:

1. record Windows, GPU, driver/runtime, vendor, VRAM and device index evidence;
2. refuse CPU fallback;
3. run the canonical Khushi GPU vector;
4. run the production-memory allocation gate using the 2 GiB dataset policy plus cache/reserve requirements;
5. run the frozen production boundary vectors;
6. autotune only bounded, architecture-safe launch candidates;
7. select the fastest safe candidate;
8. run that selected profile for the full requested benchmark duration, which is 60 seconds in the formal evidence command;
9. report selected launch geometry, elapsed seconds, hash count and H/s;
10. collect available telemetry without making telemetry absence a consensus result;
11. start the same-revision Go staging verifier on `127.0.0.1:28646` only;
12. fetch one explicit staging challenge;
13. solve it on the physical GPU;
14. submit the nonce to the independent Go verifier;
15. require `local-staging-gate=accepted`;
16. stop the verifier and write a tamper-evident evidence manifest.

## OpenCL launch safety

OpenCL tuning must bound candidate local sizes by the specific compiled `khushi_search` kernel's `CL_KERNEL_WORK_GROUP_SIZE`, not only `CL_DEVICE_MAX_WORK_GROUP_SIZE`. If no safe candidate remains, the benchmark must fail closed.

## Benchmark semantics

`--benchmark N` means:

- bounded autotuning is performed first using short candidate windows;
- the winning launch profile is then measured in a separate final benchmark window for at least `N` seconds;
- the final `Khushi Algorithm benchmark ...` line reports the final selected-profile run, not the short autotune sample;
- `autotune-selected ...` reports the selected geometry and candidate sample result separately.

For issue #24 physical evidence, `N` is 60.

## Evidence and provenance

Every backend build must record `source_revision=$GITHUB_SHA`. The combined package must verify that the NVIDIA and OpenCL artifacts were produced by successful workflow runs for the exact same source revision.

The release tag is `khushi-hardware-test-v0.2.0`. It is immutable: automation may create it once, but must never force-move or retarget an existing tag. If the tag already exists at a different revision, publication must fail closed.

## Safety invariants

This package does not authorize or activate mining. The implementation must preserve:

- `MainnetLaunchAuthorized = false`
- `MainnetMiningAuthorized = false`
- `MainnetGenesisTimestamp = 0`
- `PhysicalGPUMiningEvidenceComplete = false` until real retained physical evidence is reviewed
- `PublicCommunitySecurityReviewComplete = false`
- default network remains `sudharma-testnet-1`
- localhost staging creates no block
- Seed-1/Seed-2 and live AWS resources remain untouched

`--mine` remains gated. A successful local staging run is hardware/interoperability evidence only.

## Reference miner lessons used

The package design borrows architecture principles, not algorithm code, from mature miners: keep GPU search separate from network/session management, interrupt stale work promptly, divide search work deterministically, expose device/runtime telemetry, and independently verify GPU-produced solutions before submission.