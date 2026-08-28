param(
    [string]$MinerPath = ".\khushi-miner-nvidia.exe",
    [int]$Device = 0,
    [int]$BenchmarkSeconds = 60,
    [switch]$AllowMining
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Invoke-KhushiStep {
    param([string]$Name, [scriptblock]$Command)
    Write-Host "`n=== $Name ==="
    & $Command
    if ($LASTEXITCODE -ne 0) { throw "$Name failed with exit code $LASTEXITCODE" }
}

$MinerPath = (Resolve-Path $MinerPath).Path
$MinerDir = Split-Path -Parent $MinerPath
$MinerName = Split-Path -Leaf $MinerPath
$ChecksumPath = Join-Path $MinerDir "SHA256SUMS.txt"
$MetadataPath = Join-Path $MinerDir "build-metadata.txt"
$LogPath = Join-Path $MinerDir ("khushi-hardware-test-{0}.log" -f (Get-Date -Format "yyyyMMdd-HHmmss"))

Start-Transcript -Path $LogPath | Out-Null
try {
    Write-Host "Khushi Algorithm hardware interoperability test"
    Write-Host "miner=$MinerPath"
    Write-Host "device=$Device"
    Write-Host "benchmark_seconds=$BenchmarkSeconds"
    if (-not (Test-Path $ChecksumPath)) { throw "SHA256SUMS.txt not found beside miner: $ChecksumPath" }
    $checksumLine = Get-Content $ChecksumPath | Where-Object { $_ -match [regex]::Escape($MinerName) } | Select-Object -First 1
    if (-not $checksumLine) { throw "No checksum entry for $MinerName in SHA256SUMS.txt" }
    $expected = (($checksumLine -split '\s+')[0]).ToLowerInvariant()
    $actual = (Get-FileHash -Algorithm SHA256 $MinerPath).Hash.ToLowerInvariant()
    Write-Host "expected_sha256=$expected"
    Write-Host "actual_sha256=$actual"
    if ($expected -ne $actual) { throw "SHA256 checksum mismatch; refusing to execute miner" }
    Write-Host "checksum=ok"
    if (Test-Path $MetadataPath) { Write-Host "`n=== Build metadata ==="; Get-Content $MetadataPath }
    if (Get-Command nvidia-smi -ErrorAction SilentlyContinue) { Invoke-KhushiStep "NVIDIA driver/GPU report (nvidia-smi)" { nvidia-smi } }
    Invoke-KhushiStep "GPU discovery (--list-devices)" { & $MinerPath --list-devices }
    Invoke-KhushiStep "Canonical hardware vector (--vector-self-test)" { & $MinerPath --device $Device --vector-self-test }
    Invoke-KhushiStep "GPU benchmark (--benchmark)" { & $MinerPath --device $Device --benchmark $BenchmarkSeconds }
    if (Get-Command nvidia-smi -ErrorAction SilentlyContinue) { Invoke-KhushiStep "GPU telemetry" { & $MinerPath --device $Device --telemetry } }
    Write-Host "`nhardware-vector-and-benchmark=passed"
    Write-Host "This result is evidence for the hardware interoperability gate; it does not activate network mining or consensus."
    if ($AllowMining) {
        Write-Host "`nAllowMining was explicitly supplied. Invoking the miner's gated --mine path."
        & $MinerPath --device $Device --mine
        if ($LASTEXITCODE -ne 0) { Write-Host "mine_path_exit_code=$LASTEXITCODE"; Write-Host "A gated refusal is expected until production cache/DAG policy and staged network mining are enabled." }
    } else {
        Write-Host "network-mining=not-requested"
        Write-Host "To exercise the gated --mine command only after authorization, rerun with -AllowMining."
    }
}
finally {
    Stop-Transcript | Out-Null
    Write-Host "hardware_test_log=$LogPath"
}
