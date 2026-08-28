# Khushi Algorithm GPU Miner

Khushi Algorithm is the human-facing name for Sudharma's GPU-oriented proof-of-work. The consensus-visible protocol identifier remains `sudharma-gpupow-v1`; branding does not change hashes, vectors, activation rules, or block serialization.

## GPU backends

The miner is designed for general-purpose discrete GPUs rather than one model. NVIDIA devices use the CUDA backend and AMD or other compatible GPU devices use the OpenCL backend. The office RTX 2060 is the first available physical validation device, not a hard-coded requirement. Device selection is by runtime index and capability discovery.

There is no supported CPU mining mode. CPU fallback is explicitly prohibited: a missing or unsupported GPU must fail clearly instead of silently moving proof-of-work onto the CPU.

## Memory eligibility

GPU memory eligibility is dynamic. The miner must compare available VRAM with the actual epoch cache/DAG allocation, runtime buffers, and a safety reserve. Do not interpret the office test as a promise that every nominal 4 GB card is usable. A public minimum-VRAM claim is valid only after the production cache/DAG sizing and lifecycle are frozen and the resulting allocation is measured.

## Hardware interoperability gate

A GPU is eligible for controlled mining only after all of the following hold:

1. Artifact provenance and SHA-256 checksum verify.
2. The backend detects the intended physical GPU and driver/runtime.
3. The canonical hardware vector self-test produces exactly the locked Go reference digest.
4. A fixed-duration benchmark runs on the selected GPU and reports hashrate.
5. Available telemetry is captured. NVIDIA uses `nvidia-smi`; AMD/OpenCL telemetry depends on the vendor runtime/tooling.
6. A controlled staging solution is host-verified and accepted by the constrained mining submission endpoint.

The hardware interoperability gate does not itself activate Version 2 on Seed-1/Seed-2. Network activation remains a separate staged deployment gate.

## Windows NVIDIA test procedure

Download the generic `khushi-miner-nvidia-windows` GitHub Actions artifact from the current `feature/gpu-pow-v1` revision and extract all files into one folder. Keep `khushi-miner-nvidia.exe`, its CUDA runtime DLL, `SHA256SUMS.txt`, and `build-metadata.txt` together.

From PowerShell in a clean checkout of the same repository revision, run:

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\scripts\windows\test-khushi-miner.ps1 -MinerPath "C:\path\to\artifact\khushi-miner-nvidia.exe" -Device 0 -BenchmarkSeconds 60
```

The script validates `SHA256SUMS.txt` with `Get-FileHash`, records `nvidia-smi`, runs `--list-devices`, runs `--vector-self-test`, and then runs `--benchmark`. It writes a timestamped log next to the miner. Do not use `-AllowMining` for the initial office test.

If more than one NVIDIA GPU is installed, read the indices printed by `--list-devices` and repeat with the desired `-Device N` value.

The RTX 2060 test is successful only when the canonical vector says `vector-self-test=ok` and the benchmark completes without a CUDA error. Send the complete generated log for review before any controlled network mining attempt.

## AMD and OpenCL testing

The OpenCL backend enumerates GPU devices only and compiles against OpenCL 1.2-compatible primitives. It uses the same Khushi Algorithm constants, 64-round program, dataset derivation, canonical final digest, target comparison, nonce search, and stale-work boundary as the CUDA path. AMD hardware must pass the same canonical digest gate before being treated as interoperable.

OpenCL source and CI compilation are not a substitute for a real AMD hardware run. Vendor-specific Windows packaging and telemetry can be added without changing consensus.

## Network mining status

The current artifacts are hardware-test and benchmark artifacts. Their `--mine` path remains deliberately gated because the production epoch cache/DAG size and lifecycle are not yet frozen by the mining work contract, and the required physical interoperability test has not yet been supplied. The Windows test script exposes `-AllowMining` only so the explicit gated `--mine` refusal can be exercised; it does not bypass that gate.

No production CPU fallback, arbitrary minting, balance edits, or automatic Seed-1/Seed-2 activation are permitted. Legitimate rewards must arise only from blocks independently validated under the consensus rules.
