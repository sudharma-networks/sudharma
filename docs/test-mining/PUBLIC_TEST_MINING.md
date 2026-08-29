# Sudharma Public Test Mining — Khushi Algorithm

Sudharma GPU-PoW v1 is the **Khushi Algorithm** in public-facing documentation. The internal protocol identifier remains `sudharma-gpupow-v1` / `gpu-pow-v1` for compatibility.

## Current stage

This release is for **public GPU interoperability, self-test and benchmark participation**. It is not a promise of monetary value or production rewards. Public consensus activation and unrestricted live block submission remain gated until cross-implementation hardware tests pass.

Participants can currently:

- verify the released miner checksum and build provenance;
- detect supported GPUs;
- run the canonical GPU vector self-test;
- run the production memory/chunk-allocation gate;
- run the frozen production dataset boundary vectors;
- benchmark Khushi Algorithm on their GPU without submitting blocks;
- record Windows host, GPU and driver provenance automatically;
- record hashrate, temperature, power and utilization where supported;
- retain a standardized evidence directory for review;
- run the packaged localhost controlled-staging gate without mixing artifacts from different revisions;
- use a separately announced staging endpoint for controlled submission testing when enabled.

Do **not** submit wallet private keys, seed phrases, AWS credentials, API secrets or other secrets with test results.

## Windows NVIDIA / CUDA

Download the release asset `khushi-miner-nvidia-windows.zip` and extract it, for example to `C:\KhushiMiner`.

The package includes the NVIDIA miner, CUDA runtime dependency, same-revision local Go staging verifier, checksums, build metadata, hardware-test runner and local-staging runner.

Open PowerShell and run:

```powershell
nvidia-smi
cd C:\KhushiMiner
Set-ExecutionPolicy -Scope Process Bypass
$EvidenceDirectory = ".\evidence-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
.\test-khushi-miner.ps1 -MinerPath .\khushi-miner-nvidia.exe -Device 0 -BenchmarkSeconds 60 -EvidenceDirectory $EvidenceDirectory
```

This interoperability command does **not** enable network mining. The miner's unrestricted `--mine` path remains gated.

A successful run should include evidence such as:

```text
computer_name=...
windows_version=...
video_name=...
video_vendor=...
video_driver_version=...
checksum=ok
production-vector-sha256=ok
vector-self-test=ok
hardware-production-memory=passed
hardware-production-vectors=passed
hashrate_hps=...
hardware-vector-memory-and-benchmark=passed
network-submission=not-requested
hardware_test_log=...\hardware-test.log
```

The evidence directory contains `hardware-test.log` plus the released miner checksum and build-provenance files. Keep the whole directory so a hardware result can be audited later.

## Windows AMD / Intel / other OpenCL GPUs

Download `khushi-miner-opencl-windows.zip` and extract it to a folder such as `C:\KhushiMinerOpenCL`.

The package includes the OpenCL miner and kernels, same-revision local Go staging verifier, checksums, build metadata, hardware-test runner and local-staging runner. Install the GPU vendor driver that provides the OpenCL runtime, then run:

```powershell
cd C:\KhushiMinerOpenCL
Set-ExecutionPolicy -Scope Process Bypass
$EvidenceDirectory = ".\evidence-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
.\test-khushi-miner.ps1 -MinerPath .\khushi-miner-opencl.exe -Device 0 -BenchmarkSeconds 60 -EvidenceDirectory $EvidenceDirectory
```

The OpenCL backend discovers compatible GPU devices dynamically. CPU fallback is prohibited for supported production mining.

The Windows `video_adapter_ram_bytes=` value is supplemental host metadata and can be inaccurate on some drivers. The miner's `--list-devices` output and the production-memory self-test are the authoritative hardware gates for the project's 4 GiB minimum dedicated-VRAM policy.

## Evidence directory and what to report

When `-EvidenceDirectory` is supplied, the runner writes a reproducible evidence bundle containing at least:

- `hardware-test.log` — full transcript, including Windows version/build, machine information, GPU/video-controller information, miner device discovery, vector tests, memory gate and benchmark output;
- `miner-build-metadata.txt` — source revision and packaged miner build provenance;
- `miner-SHA256SUMS.txt` — released package checksums used during the test.

For a controlled staging run, the same directory also records `challenge.json`, `solution.json` and `submit-result.json` when those stages are reached.

Please report or attach the evidence directory (compressing it to a ZIP is fine) and identify:

- GPU model and dedicated VRAM reported by the miner;
- Windows version;
- NVIDIA driver/CUDA version or OpenCL vendor/runtime information available from the installed driver;
- backend used (`CUDA` or `OpenCL`);
- canonical vector self-test result;
- production memory/chunk-allocation result;
- production dataset boundary-vector result;
- benchmark duration and hashrate;
- temperature, power and utilization where available;
- whether network submission was `not-requested` or an explicitly controlled staging submission was accepted.

Public result thread:

https://github.com/sudharma-networks/sudharma/issues/24

## Controlled staging mining

Controlled staging is separate from unrestricted network mining. Do not point a miner at arbitrary RPC endpoints and do not expose administrative RPC services to the public internet.

### Packaged localhost staging gate — recommended hardware interoperability check

Each released Windows miner ZIP now contains its matching miner, the same-revision independent Go staging verifier, checksums and both PowerShell runners. From a freshly extracted NVIDIA CUDA **or** OpenCL package, run:

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\run-local-staging-gate.ps1 -Device 0 -BenchmarkSeconds 60
```

The wrapper auto-detects the packaged CUDA or OpenCL miner, verifies the staging verifier checksum, starts the verifier on localhost only, runs the full hardware/vector/memory/benchmark suite, fetches a bounded staging challenge, submits the GPU-produced nonce to the independent Go verifier, creates an auditable evidence directory, and stops the verifier afterward.

A successful run ends with:

```text
local-staging-gate=accepted
```

This localhost test creates **no block**, does not touch Seed-1 or Seed-2, and does not activate consensus mining.

### Separately announced staging endpoint

When a Sudharma staging mining endpoint is explicitly announced and approved for testing, the hardware runner can perform the bounded staging flow with both `-SubmitStagingSolution` and `-StagingEndpoint`:

```powershell
$EvidenceDirectory = ".\staging-evidence-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
.\test-khushi-miner.ps1 -MinerPath .\khushi-miner-nvidia.exe -Device 0 -BenchmarkSeconds 60 -EvidenceDirectory $EvidenceDirectory -SubmitStagingSolution -StagingEndpoint https://APPROVED-STAGING-BASE
```

Use only the staging base URL supplied by the Sudharma test procedure. For the packaged localhost staging gate, use its `run-local-staging-gate.ps1` wrapper instead of inventing an endpoint.

A controlled submission is considered successful only when the independently implemented Go staging verifier returns `accepted`. The evidence bundle should then contain `submit-result.json` showing that result. No block is created by this isolated staging flow and consensus is not activated.

## Network activation safety

Seed-1/Seed-2 GPU-PoW consensus activation is intentionally not enabled merely by downloading or running this release. The activation gate still requires deterministic Go vectors, cross-vendor GPU interoperability, no CPU production fallback, real GPU-mined Version-2 validation on both seed nodes, and the later mined-funds → faucet → Android wallet end-to-end test.

Until those gates are deliberately completed, do not use unrestricted `--mine` and do not treat a benchmark or staging acceptance as authorization to change seed-node consensus.

## Source

Development branch:

https://github.com/sudharma-networks/sudharma/tree/feature/gpu-pow-v1

Repository:

https://github.com/sudharma-networks/sudharma
