# Khushi Algorithm — RTX 2060 Hardware Interoperability Gate

This procedure is the required NVIDIA hardware gate before Sudharma GPU-PoW v1 can be enabled on Seed-1/Seed-2. It tests the real CUDA execution path but does **not** activate consensus or submit live blocks.

## Prerequisites

- Windows 10/11 64-bit office PC with an NVIDIA GeForce RTX 2060.
- Current NVIDIA display driver installed and `nvidia-smi` available in PowerShell.
- The GitHub Actions artifact named `khushi-miner-rtx2060-windows` from the latest successful **Khushi RTX 2060 Windows CUDA** workflow on `feature/gpu-pow-v1`.
- Extract the artifact to a new empty directory. The artifact contains the miner executable, CUDA runtime DLL, source-revision metadata, and `SHA256SUMS.txt`.

The executable is built specifically for Turing compute capability `sm_75`. No CPU mining fallback is allowed.

## 1. Open PowerShell in the extracted artifact directory

```powershell
Get-Location
Get-ChildItem
```

Confirm that `khushi-miner-rtx2060.exe`, `SHA256SUMS.txt`, `build-metadata.txt`, and a `cudart64_*.dll` file are present.

## 2. Verify artifact provenance and checksum

```powershell
Get-Content .\build-metadata.txt
Get-Content .\SHA256SUMS.txt
(Get-FileHash .\khushi-miner-rtx2060.exe -Algorithm SHA256).Hash.ToLower()
```

The displayed SHA-256 must exactly match the first value in `SHA256SUMS.txt`. Stop the gate if it does not match.

## 3. Confirm the NVIDIA driver and RTX 2060

```powershell
nvidia-smi
.\khushi-miner-rtx2060.exe --device-info
```

The miner must report a CUDA device and the expected RTX 2060-class device with compute capability 7.5. Stop if CUDA reports no device or a driver/runtime error.

## 4. Run the canonical Go-vs-CUDA vector self-test

```powershell
.\khushi-miner-rtx2060.exe --vector-self-test
```

Required success output includes:

```text
vector-digest=2a7c15fc6c84a67d43ff7074ac5835aa433145f89d10d1d9e36a99fe22da4b2b
vector-self-test=ok
```

This executes the header binding, 16 programmatic lanes, dataset accesses, lane reduction, and final digest on the NVIDIA GPU using the locked 8-node interoperability fixture. Stop immediately if the digest differs.

## 5. Record temperature, power and utilization

```powershell
.\khushi-miner-rtx2060.exe --telemetry
```

For continuous observation during the benchmark, open a second PowerShell window and run:

```powershell
nvidia-smi --query-gpu=name,driver_version,temperature.gpu,power.draw,utilization.gpu,memory.used --format=csv -l 1
```

Leave it running until the benchmark finishes, then press `Ctrl+C`.

## 6. Run the 60-second GPU mining benchmark

In the first PowerShell window:

```powershell
.\khushi-miner-rtx2060.exe --benchmark 60
```

Record the complete output, especially `hashrate_hps`. This benchmark uses the real CUDA nonce-search kernel and does not submit blocks.

## 7. Save one evidence log

```powershell
$log = "khushi-rtx2060-gate.txt"
"=== NVIDIA SMI ===" | Set-Content $log
nvidia-smi | Add-Content $log
"=== BUILD METADATA ===" | Add-Content $log
Get-Content .\build-metadata.txt | Add-Content $log
"=== CHECKSUM ===" | Add-Content $log
Get-Content .\SHA256SUMS.txt | Add-Content $log
"=== DEVICE ===" | Add-Content $log
.\khushi-miner-rtx2060.exe --device-info 2>&1 | Add-Content $log
"=== VECTOR ===" | Add-Content $log
.\khushi-miner-rtx2060.exe --vector-self-test 2>&1 | Add-Content $log
"=== TELEMETRY ===" | Add-Content $log
.\khushi-miner-rtx2060.exe --telemetry 2>&1 | Add-Content $log
"=== BENCHMARK 60s ===" | Add-Content $log
.\khushi-miner-rtx2060.exe --benchmark 60 2>&1 | Add-Content $log
Get-Content $log
```

Return the complete `khushi-rtx2060-gate.txt` output for review.

## Live mining remains gated

Do not treat `--benchmark` as a live-network mining command. `--mine` intentionally refuses to start until this RTX 2060 interoperability evidence is reviewed and the production epoch-cache policy plus staged testnet deployment gates are complete. Seed-1/Seed-2 activation must remain disabled until those gates pass.
