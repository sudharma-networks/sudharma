# Khushi Algorithm Multi-Vendor GPU Miner Design

## Purpose

Provide a production-oriented external miner for Sudharma GPU-PoW v1, branded **Khushi Algorithm**, without changing the consensus-visible algorithm identifier `sudharma-gpupow-v1` or any already-frozen interoperability vectors.

## Hardware support model

The miner uses one shared work/template, DAG/cache, nonce-range, stale-work and target-validation contract with multiple GPU execution backends:

- NVIDIA: CUDA backend.
- AMD and other compatible devices: OpenCL backend.
- CPU mining is not a supported fallback and must never activate silently.

The office RTX 2060 is the first physical interoperability device only; no RTX-2060-specific consensus or search behavior may be encoded.

The public compatibility target is discrete GPUs with at least 4 GiB of
dedicated VRAM. NVIDIA devices use CUDA, while AMD and other vendors use an
OpenCL 1.2-or-newer GPU runtime. NVIDIA devices may also use the OpenCL
artifact when their driver exposes a compatible runtime. A GPU brand, model,
or advertised memory label is never a consensus input.

## Memory eligibility

Device eligibility is derived from the actual epoch DAG/cache allocation plus runtime buffers and a safety reserve. The miner reports `required_vram_bytes` and `available_vram_bytes` and refuses a device that cannot satisfy the allocation. Documentation may advertise 4 GB-class compatibility only after the frozen production DAG configuration fits within the real usable allocation on such a device.

“4 GiB supported” means that the frozen allocation must fit on a conforming
4 GiB device under both backends. Runtime preflight remains mandatory:
driver-reserved memory, unsupported CUDA/OpenCL capabilities, or an allocation
failure must produce an explicit refusal. Cards with more than 4 GiB follow
exactly the same work and verification contract.

## Shared mining contract

All GPU backends consume the same immutable work identity, canonical header prefix, nonce range, target, height/program seed and epoch data. They must produce exactly the same final digest as the Go light verifier for every canonical vector.

Nonce dispatch is explicit as `(nonce_start, nonce_count)`. A monotonic work generation is supplied to each dispatch. Kernels periodically compare the dispatch generation with `stale_generation`; work from an obsolete generation must stop without submitting a solution.

The first valid nonce is reported atomically. Host code always re-verifies the returned nonce with the canonical reference/verifier path before submission.

## CUDA backend

CUDA owns NVIDIA device discovery, memory allocation, epoch data upload/build, kernel launch, synchronization and telemetry. CUDA kernels use the existing verified CUDA-compatible primitives, dataset generation, 64-round lane program, group reduction and final digest contract.

## OpenCL backend

OpenCL implements the same primitive contract in OpenCL C, uses unchanged vectors, and exposes the same device-selection and nonce-dispatch semantics. Vendor differences are isolated to backend code; consensus inputs and outputs are identical.

## CLI behavior

The miner executable exposes human-facing `Khushi Algorithm` branding while keeping protocol identifiers unchanged. Required operating modes:

- `--list-devices`: enumerate supported GPU devices and memory/capability details.
- `--self-test`: run deterministic primitive/canonical-vector checks.
- `--benchmark`: run bounded mining work without submitting blocks and report hashrate.
- `--mine`: external mining mode using the Sudharma mining work API; GPU backend is mandatory.
- `--device N`: choose a specific detected GPU.

No command may silently switch to CPU mining.

## Telemetry

Common output includes backend, device name, driver/runtime, total/usable VRAM, required VRAM, hashes attempted, H/s, accepted/rejected/stale counts and runtime. Temperature, power and utilization are reported when the vendor tooling/runtime exposes them; absence of optional telemetry must not alter consensus behavior.

## Build and artifact model

Windows CI produces an NVIDIA CUDA artifact first because the available physical gate device is an RTX 2060. Artifact metadata includes source revision and SHA-256 checksum. OpenCL-capable Windows artifacts follow the same provenance rules.

A build artifact is not an interoperability pass. Hardware gating requires the executable to run on a real GPU, pass canonical vectors, complete a fixed-duration benchmark and, only against a non-production/staging mining endpoint, submit a controlled solution that the Go verifier independently accepts.

## Deployment gates

Seed-1/Seed-2 consensus activation remains prohibited until:

1. Go reference vectors pass.
2. CUDA matches all canonical vectors.
3. The real CUDA search path passes on the RTX 2060 or another approved NVIDIA GPU.
4. Returned GPU nonces are independently accepted by the Go verifier.
5. No CPU production fallback exists.
6. Staged two-node activation checks are explicitly performed.

Multi-vendor support must not weaken those gates. OpenCL interoperability is an additional backend gate, not a reason to change consensus.
