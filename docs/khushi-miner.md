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

## Windows prerequisites

For the first NVIDIA hardware interoperability run, use a 64-bit Windows 10 or Windows 11 machine with a supported NVIDIA GPU, a working NVIDIA driver, PowerShell 5.1 or PowerShell 7+, and the revision-matched `khushi-miner-nvidia-windows` artifact. Keep the miner executable, its packaged CUDA runtime DLL where present, `SHA256SUMS.txt`, and `build-metadata.txt` in the same extracted folder.

Use a clean checkout of `sudharma-networks/sudharma` at the same source revision recorded in the artifact metadata. `nvidia-smi` should be available for the NVIDIA driver report and optional telemetry. The script is benchmark only by default: it verifies provenance, enumerates the GPU, executes the canonical vector self-test, runs a bounded benchmark, and records telemetry without requesting network submission.

## Windows NVIDIA test procedure

Download the generic `khushi-miner-nvidia-windows` GitHub Actions artifact from the current `feature/gpu-pow-v1` revision and extract all files into one folder.

From PowerShell in the matching repository checkout, run:

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\scripts\windows\test-khushi-miner.ps1 -MinerPath "C:\path\to\artifact\khushi-miner-nvidia.exe" -Device 0 -BenchmarkSeconds 60
```

The script validates `SHA256SUMS.txt` with `Get-FileHash`, records `nvidia-smi`, runs `--list-devices`, runs `--vector-self-test`, runs `--benchmark`, and requests `--telemetry` when NVIDIA tooling is present. It writes a timestamped log next to the miner and reports `network-submission=not-requested` in the default path.

If more than one NVIDIA GPU is installed, read the indices printed by `--list-devices` and repeat with the desired `-Device N` value.

The RTX 2060 test is successful only when the canonical vector says `vector-self-test=ok` and the benchmark completes without a CUDA error. Preserve the complete generated log because the interoperability evidence must include the artifact checksum, GPU model/driver/VRAM, vector result, benchmark H/s, runtime, and available telemetry.

### Controlled staging submission gate

Do not point the hardware test at Seed-1, Seed-2, or any production/public consensus endpoint. A controlled solution attempt requires both the explicit `-SubmitStagingSolution` switch and a pathless HTTP(S) mining base URL supplied through `-StagingEndpoint`, for example:

```powershell
.\scripts\windows\test-khushi-miner.ps1 -MinerPath "C:\path\to\artifact\khushi-miner-nvidia.exe" -Device 0 -BenchmarkSeconds 60 -SubmitStagingSolution -StagingEndpoint "https://staging-mining.example"
```

Supplying only `-SubmitStagingSolution` is rejected. The staging URL must be an absolute `http` or `https` base URL with no path, query, or fragment, matching the constrained `/v1/mining/work` and `/v1/mining/submit` client boundary.

The current CUDA artifact still deliberately gates `--mine`; therefore an exit through its gated refusal is expected until the physical hardware interoperability requirements and the remaining staged-mining integration are satisfied. The PowerShell flag records explicit authorization and the staging endpoint but does not bypass the miner gate or activate consensus. Any future successful staging submission must still be independently re-verified by the host/Go verifier before it counts as interoperability evidence.

## AMD and OpenCL testing

The OpenCL backend enumerates GPU devices only and compiles against OpenCL 1.2-compatible primitives. It uses the same Khushi Algorithm constants, 64-round program, dataset derivation, canonical final digest, target comparison, nonce search, and stale-work boundary as the CUDA path. AMD hardware must pass the same canonical digest gate before being treated as interoperable.

OpenCL source and CI compilation are not a substitute for a real AMD hardware run. Vendor-specific Windows packaging and telemetry can be added without changing consensus.

## Network mining status

The current artifacts remain hardware-test and benchmark artifacts. Their `--mine` path is deliberately gated until physical interoperability evidence is supplied and the controlled staging path is approved. The Windows test script defaults to no network submission and requires both `-SubmitStagingSolution` and `-StagingEndpoint` before it will even invoke that gated path.

No production CPU fallback, arbitrary minting, balance edits, or automatic Seed-1/Seed-2 activation are permitted. Legitimate rewards must arise only from blocks independently validated under the consensus rules.
