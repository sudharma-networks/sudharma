# Khushi GPU Autotune and Evidence Policy

## Purpose

Khushi must run across compatible NVIDIA, AMD and other discrete GPUs without treating the office RTX 2060 as a performance template. The miner therefore separates **hardware facts**, **advisory tuning priors**, **local Khushi measurements**, and **physical interoperability evidence**.

GPU model names, architecture families, published hashrates and tuning profiles are never consensus inputs. Consensus-visible work, target, nonce and digest behavior remain identical across backends.

## Runtime-first policy

The miner uses runtime-reported capabilities before any model-name inference:

- CUDA: compute capability, multiprocessor count, maximum threads per block and available/total VRAM.
- OpenCL: device vendor, compute-unit count, maximum work-group size, global memory and maximum allocation size.
- Production memory preflight remains authoritative. A known GPU model is not enough to bypass an allocation failure.
- Unknown vendors receive conservative candidates rather than a CPU fallback.

The shared launch policy generates a small bounded set of candidates. `--benchmark` measures those candidates using Khushi itself and selects the fastest stable geometry on the installed GPU/driver/runtime.

## Architecture references

The initial family labels are based on vendor documentation, not mining-site model tables:

- NVIDIA CUDA GPU Compute Capability: https://developer.nvidia.com/cuda/gpus
  - RTX 20 / Turing consumer devices are compute capability 7.5.
  - RTX 30 / Ampere consumer devices are compute capability 8.6.
  - RTX 40 / Ada consumer devices are compute capability 8.9.
  - RTX 50 / Blackwell consumer devices are compute capability 12.0 in NVIDIA's current table.
- AMD ROCm GPU specifications: https://rocm.docs.amd.com/en/latest/reference/gpu-specs.html
- AMD ROCm compatibility matrix: https://rocm.docs.amd.com/en/latest/compatibility/compatibility-matrix.html
  - Current Radeon RX 7000 family entries map to RDNA 3 targets such as `gfx1100`, `gfx1101` and `gfx1102`.
  - Current Radeon RX 9000 family entries map to RDNA 4 targets such as `gfx1200` and `gfx1201`.

These mappings select only a **starting candidate set**. They do not determine Khushi H/s.

## External mining-platform data

Mining platforms such as Kryptex and Hashrate.no publish per-GPU measurements for many algorithms. These data are useful for understanding broad hardware behavior and for deciding which launch sizes are worth testing, but they are not transferable Khushi benchmarks.

Example reference:

- Kryptex RTX 4090 hardware page: https://www.kryptex.com/en/hardware/nvidia-rtx-4090

The same GPU has materially different rates and power behavior across KawPoW, Octopus, Autolykos and other algorithms. Therefore:

1. Do not copy another algorithm's H/s into a Khushi table.
2. Do not rank Khushi GPUs solely from external mining data.
3. Do not require a third-party API or internet connection for mining startup.
4. Use external data only to seed safe candidate choices and documentation.
5. Report Khushi H/s only after the local Khushi benchmark actually measures it.

## Evidence classes

### `external-profile`

The vendor/family/runtime facts are known from public documentation or the installed runtime. This state says nothing about measured Khushi performance or staging acceptance.

### `locally-benchmarked`

The exact installed GPU completed the Khushi self-test/benchmark and produced a measured local H/s with a selected launch geometry. This is a performance observation for that GPU, driver and miner revision. It is not automatically portable to another card of the same model.

### `physically-verified`

The retained project hardware-evidence procedure passed: canonical vectors, memory policy, benchmark/telemetry where required, independent host verification and the applicable controlled staging round trip. Only this class can satisfy the formal physical GPU evidence gate.

## Safety boundaries

- CPU mining fallback remains prohibited.
- ASIC claims are not made by the autotuner.
- GPU-PoW network activation remains disabled unless separately authorized through the existing activation process.
- RTX 2060 evidence remains RTX 2060 evidence; adding an architecture profile does not broaden it to other NVIDIA models.
- AMD/OpenCL remains pending physical evidence until an actual qualifying non-NVIDIA GPU completes the retained gate.
- Autotune must never alter a consensus-visible header, target, nonce interpretation, digest function, epoch dataset rule or reward address.

## Result reporting

A benchmark should report at least:

- backend and selected device,
- runtime family label,
- candidate local/work-group size,
- bounded work items per launch,
- measured elapsed seconds and hashes,
- measured Khushi H/s,
- final `autotune-selected` geometry.

A result should be keyed to the miner source revision and driver/runtime when retained as evidence. Published external mining numbers must remain explicitly labeled as external/reference data.
