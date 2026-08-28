# Sudharma Public Test Mining — Khushi Algorithm

Sudharma GPU-PoW v1 is the **Khushi Algorithm** in public-facing documentation. The internal protocol identifier remains `sudharma-gpupow-v1` / `gpu-pow-v1` for compatibility.

## Current stage

This release is for **public GPU interoperability, self-test and benchmark participation**. It is not a promise of monetary value or production rewards. Public consensus activation and unrestricted live block submission remain gated until cross-implementation hardware tests pass.

Participants can currently:

- verify the released miner checksum and build provenance;
- detect supported GPUs;
- run the canonical GPU vector self-test;
- benchmark Khushi Algorithm on their GPU without submitting blocks;
- record hashrate, temperature, power and utilization where supported;
- publish their hardware results for the project;
- use a separately announced staging endpoint for controlled submission testing when enabled.

Do **not** submit wallet private keys, seed phrases, AWS credentials, API secrets or other secrets with test results.

## Windows NVIDIA / CUDA

Download the release asset `khushi-miner-nvidia-windows.zip` and extract it, for example to `C:\KhushiMiner`.

The package includes the NVIDIA miner, CUDA runtime dependency, checksums, build metadata and the hardware-test PowerShell runner.

Open PowerShell and run:

```powershell
nvidia-smi
cd C:\KhushiMiner
Set-ExecutionPolicy -Scope Process Bypass
.\test-khushi-miner.ps1 -MinerPath .\khushi-miner-nvidia.exe -Device 0 -BenchmarkSeconds 60
```

Do not add `-AllowMining` during the interoperability test.

A successful run should include evidence such as:

```text
checksum=ok
Khushi Algorithm CUDA devices=...
vector-self-test=ok
hashrate_hps=...
hardware-vector-and-benchmark=passed
```

The runner writes a `khushi-hardware-test-*.log` file. Post the result to public issue #24:

https://github.com/sudharma-networks/sudharma/issues/24

## Windows AMD / Intel / other OpenCL GPUs

Download `khushi-miner-opencl-windows.zip` and extract it to a folder such as `C:\KhushiMinerOpenCL`.

Make sure the GPU vendor driver that provides an OpenCL runtime is installed. Then run:

```powershell
cd C:\KhushiMinerOpenCL
Set-ExecutionPolicy -Scope Process Bypass
.\test-khushi-miner.ps1 -MinerPath .\khushi-miner-opencl.exe -Device 0 -BenchmarkSeconds 60
```

The OpenCL backend is intended to discover compatible GPU devices dynamically. CPU fallback is prohibited for supported production mining.

## What to report

Please post:

- GPU model and VRAM;
- Windows version;
- NVIDIA driver/CUDA version or OpenCL platform/driver;
- backend used (`CUDA` or `OpenCL`);
- canonical vector self-test result;
- benchmark duration and hashrate;
- temperature, power and utilization where available;
- the generated hardware-test log when possible.

Public result thread:

https://github.com/sudharma-networks/sudharma/issues/24

## Controlled staging mining

The miner and external work API are being prepared for controlled test submissions. Do not point a miner at arbitrary RPC endpoints and do not expose administrative RPC services to the public internet.

When a Sudharma staging mining endpoint is explicitly announced, participants may be asked to run a controlled solution submission. That submission is only considered valid after the candidate independently matches the Go consensus verifier.

## Network activation safety

Seed-1/Seed-2 GPU-PoW consensus activation is intentionally not enabled merely by downloading this release. The release gate requires deterministic Go vectors, GPU interoperability, no CPU production fallback, real GPU-mined Version-2 validation on both seed nodes, and the later mined-funds → faucet → Android wallet end-to-end test.

## Source

Development branch:

https://github.com/sudharma-networks/sudharma/tree/feature/gpu-pow-v1

Repository:

https://github.com/sudharma-networks/sudharma
