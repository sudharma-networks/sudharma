param(
    [string]$MinerPath = ".\khushi-miner-nvidia.exe",
    [int]$Device = 0,
    [int]$BenchmarkSeconds = 60,
    [string]$StagingEndpoint = "",
    [switch]$SubmitStagingSolution
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Invoke-KhushiStep {
    param([string]$Name, [scriptblock]$Command)
    Write-Host "`n=== $Name ==="
    & $Command
    if ($LASTEXITCODE -ne 0) { throw "$Name failed with exit code $LASTEXITCODE" }
}

if ($SubmitStagingSolution -and [string]::IsNullOrWhiteSpace($StagingEndpoint)) {
    throw "SubmitStagingSolution requires -StagingEndpoint"
}

if (-not [string]::IsNullOrWhiteSpace($StagingEndpoint)) {
    $stagingUri = $null
    if (-not [Uri]::TryCreate($StagingEndpoint, [UriKind]::Absolute, [ref]$stagingUri)) {
        throw "StagingEndpoint must be an absolute HTTP(S) base URL"
    }
    if ($stagingUri.Scheme -notin @("http", "https") -or [string]::IsNullOrWhiteSpace($stagingUri.Host)) {
        throw "StagingEndpoint must be an absolute HTTP(S) base URL"
    }
    if ($stagingUri.AbsolutePath -ne "/" -or -not [string]::IsNullOrWhiteSpace($stagingUri.Query) -or -not [string]::IsNullOrWhiteSpace($stagingUri.Fragment)) {
        throw "StagingEndpoint must not include a path, query, or fragment"
    }
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

    if ($SubmitStagingSolution) {
        $normalizedEndpoint = $StagingEndpoint.TrimEnd("/")
        $env:SUDHARMA_MINING_ENDPOINT = $normalizedEndpoint
        Write-Host "`nnetwork-submission=explicitly-requested"
        Write-Host "staging_endpoint=$normalizedEndpoint"
        Write-Host "Invoking the miner's gated --mine path only after an explicit staging endpoint was supplied."
        & $MinerPath --device $Device --mine
        $mineExitCode = $LASTEXITCODE
        if ($mineExitCode -eq 3) {
            Write-Host "staging-submit=gated"
            Write-Host "The current CUDA artifact still refuses network mining until the remaining hardware interoperability gate is satisfied."
        } elseif ($mineExitCode -ne 0) {
            throw "staging submission path failed with exit code $mineExitCode"
        } else {
            Write-Host "staging-submit=completed"
        }
    } else {
        Write-Host "network-submission=not-requested"
        Write-Host "Benchmark/self-test mode is the default. Controlled submission requires both -SubmitStagingSolution and -StagingEndpoint."
    }
}
finally {
    Stop-Transcript | Out-Null
    Write-Host "hardware_test_log=$LogPath"
}
