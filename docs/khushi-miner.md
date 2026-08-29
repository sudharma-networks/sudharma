# Khushi Algorithm GPU Miner

Khushi Algorithm is the human-facing name for Sudharma's GPU-oriented proof-of-work. The consensus-visible protocol identifier remains `sudharma-gpupow-v1`; branding does not change hashes, vectors, activation rules, or block serialization.

## GPU backends

The miner is designed for general-purpose discrete GPUs rather than one model.
The compatibility target is 4 GiB or more of dedicated VRAM. NVIDIA devices
use the CUDA backend; AMD and other compatible GPU devices use an OpenCL
1.2-or-newer backend. NVIDIA may also use OpenCL when its installed driver
provides a compatible runtime. The office RTX 2060 is the first available
physical validation device, not a hard-coded requirement. Device selection is
by runtime index and capability discovery.

There is no supported CPU mining mode. CPU fallback is explicitly prohibited: a missing or unsupported GPU must fail clearly instead of silently moving proof-of-work onto the CPU.

## Memory eligibility

GPU memory eligibility is dynamic. The miner must compare available VRAM with
the actual epoch cache/DAG allocation, runtime buffers, and a safety reserve.
The production sizing must fit within measured usable memory on supported 4 GiB
CUDA and OpenCL cards before the compatibility target is declared validated.
A nominal 4 GB/4 GiB label alone is insufficient when the installed driver
lacks the required runtime, reserves too much memory, or the actual allocation
fails. Devices above 4 GiB use the same algorithm and receive no consensus
advantage.

## Hardware interoperability gate

A GPU is eligible for controlled mining only after all of the following hold:

1. Artifact provenance and SHA-256 checksum verify.
2. The backend detects the intended physical GPU and driver/runtime.
3. The canonical hardware vector self-test produces exactly the locked Go reference digest.
4. A fixed-duration benchmark runs on the selected GPU and reports hashrate.
5. Available telemetry is captured. NVIDIA uses `nvidia-smi`; AMD/OpenCL telemetry depends on the vendor runtime/tooling.
6. A controlled staging solution is independently accepted by the isolated Go staging verifier.

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

Do not point the hardware test at Seed-1, Seed-2, or any live consensus/mining endpoint. A controlled solution attempt requires both the explicit `-SubmitStagingSolution` switch and a pathless HTTP(S) base URL for the isolated `sudharma-gpupow-staging` verifier, supplied through `-StagingEndpoint`, for example:

```powershell
.\scripts\windows\test-khushi-miner.ps1 -MinerPath "C:\path\to\artifact\khushi-miner-nvidia.exe" -Device 0 -BenchmarkSeconds 60 -SubmitStagingSolution -StagingEndpoint "https://staging-mining.example"
```

Supplying only `-SubmitStagingSolution` is rejected. The staging URL must be an absolute `http` or `https` base URL with no path, query, or fragment. The script talks only to the staging-specific endpoints:

- `GET /v1/mining/staging/challenge`
- `POST /v1/mining/staging/submit`

After validating that the challenge is explicitly `staging=true`, algorithm `sudharma-gpupow-v1`, height `0`, and cache node count `8`, the script invokes the CUDA miner's dedicated `--staging-search` mode. That mode searches the supplied non-consensus challenge and prints `staging-solution-nonce=...` when it finds a candidate. The script then posts the exact challenge plus nonce back to the isolated staging verifier and requires status `accepted`.

A successful staging response proves that a physical GPU nonce agrees with the independent Go staging verifier. It creates no block, changes no balance, selects no production cache/DAG policy, and does not activate consensus. Normal `--mine` remains deliberately gated throughout Stage B.

The staging verifier currently keeps a bounded set of outstanding challenges so parallel challenge requests do not invalidate one another. Challenges are single-use, expire after the configured short staging TTL, and replay/mutated submissions are rejected.

## AMD and OpenCL testing

The OpenCL backend enumerates GPU devices only and compiles against OpenCL 1.2-compatible primitives. It uses the same Khushi Algorithm constants, 64-round program, dataset derivation, canonical final digest, target comparison, nonce search, and stale-work boundary as the CUDA path. AMD hardware must pass the same canonical digest gate before being treated as interoperable.

OpenCL source and CI compilation are not a substitute for a real AMD hardware run. Vendor-specific Windows packaging and telemetry can be added without changing consensus.

## Network mining status

The current artifacts remain hardware-test and controlled-staging artifacts. Their normal `--mine` path remains deliberately gated. Benchmark/self-test is the default, while controlled staging requires both `-SubmitStagingSolution` and a dedicated staging-verifier base URL.

No production CPU fallback, arbitrary minting, balance edits, or automatic Seed-1/Seed-2 activation are permitted. Legitimate rewards must arise only from blocks independently validated under the consensus rules.
