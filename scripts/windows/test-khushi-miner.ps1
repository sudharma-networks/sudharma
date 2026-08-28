param(
    [string]$MinerPath = ".\khushi-miner-nvidia.exe",
    [string]$ProductionVectorPath = "",
    [int]$Device = 0,
    [int]$BenchmarkSeconds = 60,
    [string]$StagingEndpoint = "",
    [switch]$SubmitStagingSolution,
    [string]$EvidenceDirectory = ""
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Invoke-KhushiStep {
    param([string]$Name, [scriptblock]$Command)
    Write-Host "`n=== $Name ==="
    & $Command
    if ($LASTEXITCODE -ne 0) { throw "$Name failed with exit code $LASTEXITCODE" }
}

function Write-KhushiHostEvidence {
    Write-Host "`n=== Windows host / GPU provenance ==="
    Write-Host "computer_name=$([Environment]::MachineName)"
    Write-Host "powershell_version=$($PSVersionTable.PSVersion.ToString())"

    try {
        $os = Get-CimInstance -ClassName Win32_OperatingSystem -ErrorAction Stop
        Write-Host "windows_caption=$($os.Caption)"
        Write-Host "windows_version=$($os.Version)"
        Write-Host "windows_build=$($os.BuildNumber)"
    } catch {
        Write-Warning "Unable to read Win32_OperatingSystem: $($_.Exception.Message)"
    }

    try {
        $system = Get-CimInstance -ClassName Win32_ComputerSystem -ErrorAction Stop
        Write-Host "system_manufacturer=$($system.Manufacturer)"
        Write-Host "system_model=$($system.Model)"
    } catch {
        Write-Warning "Unable to read Win32_ComputerSystem: $($_.Exception.Message)"
    }

    try {
        $videoControllers = @(Get-CimInstance -ClassName Win32_VideoController -ErrorAction Stop)
        foreach ($video in $videoControllers) {
            Write-Host "video_name=$($video.Name)"
            Write-Host "video_vendor=$($video.AdapterCompatibility)"
            Write-Host "video_driver_version=$($video.DriverVersion)"
            Write-Host "video_adapter_ram_bytes=$($video.AdapterRAM)"
        }
    } catch {
        Write-Warning "Unable to read Win32_VideoController: $($_.Exception.Message)"
    }
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

if ([string]::IsNullOrWhiteSpace($ProductionVectorPath)) {
    if ($MinerName -match "opencl") {
        $productionVectorCandidates = @(
            "khushi-production-vectors-opencl.exe",
            "khushi-production-vectors-nvidia.exe"
        )
    } else {
        $productionVectorCandidates = @(
            "khushi-production-vectors-nvidia.exe",
            "khushi-production-vectors-opencl.exe"
        )
    }
    foreach ($candidate in $productionVectorCandidates) {
        $candidatePath = Join-Path $MinerDir $candidate
        if (Test-Path $candidatePath) {
            $ProductionVectorPath = $candidatePath
            break
        }
    }
    if ([string]::IsNullOrWhiteSpace($ProductionVectorPath)) {
        throw "Production vector executable not found beside miner"
    }
}
$ProductionVectorPath = (Resolve-Path $ProductionVectorPath).Path
$ProductionVectorName = Split-Path -Leaf $ProductionVectorPath

$ResolvedEvidenceDirectory = ""
if (-not [string]::IsNullOrWhiteSpace($EvidenceDirectory)) {
    $ResolvedEvidenceDirectory = (New-Item -ItemType Directory -Force -Path $EvidenceDirectory).FullName
    $LogPath = Join-Path $ResolvedEvidenceDirectory "hardware-test.log"
} else {
    $LogPath = Join-Path $MinerDir ("khushi-hardware-test-{0}.log" -f (Get-Date -Format "yyyyMMdd-HHmmss"))
}

Start-Transcript -Path $LogPath | Out-Null
try {
    Write-Host "Khushi Algorithm hardware interoperability test"
    Write-Host "miner=$MinerPath"
    Write-Host "production_vector_executable=$ProductionVectorPath"
    Write-Host "device=$Device"
    Write-Host "benchmark_seconds=$BenchmarkSeconds"
    if (-not [string]::IsNullOrWhiteSpace($ResolvedEvidenceDirectory)) {
        Write-Host "evidence_directory=$ResolvedEvidenceDirectory"
    }

    Write-KhushiHostEvidence

    if (-not (Test-Path $ChecksumPath)) { throw "SHA256SUMS.txt not found beside miner: $ChecksumPath" }
    $checksumLine = Get-Content $ChecksumPath | Where-Object { $_ -match [regex]::Escape($MinerName) } | Select-Object -First 1
    if (-not $checksumLine) { throw "No checksum entry for $MinerName in SHA256SUMS.txt" }
    $expected = (($checksumLine -split '\s+')[0]).ToLowerInvariant()
    $actual = (Get-FileHash -Algorithm SHA256 $MinerPath).Hash.ToLowerInvariant()
    Write-Host "expected_sha256=$expected"
    Write-Host "actual_sha256=$actual"
    if ($expected -ne $actual) { throw "SHA256 checksum mismatch; refusing to execute miner" }
    Write-Host "checksum=ok"

    $productionVectorChecksumLine = Get-Content $ChecksumPath | Where-Object { $_ -match [regex]::Escape($ProductionVectorName) } | Select-Object -First 1
    if (-not $productionVectorChecksumLine) { throw "No checksum entry for $ProductionVectorName in SHA256SUMS.txt" }
    $productionVectorExpected = (($productionVectorChecksumLine -split '\s+')[0]).ToLowerInvariant()
    $productionVectorActual = (Get-FileHash -Algorithm SHA256 $ProductionVectorPath).Hash.ToLowerInvariant()
    Write-Host "production_vector_expected_sha256=$productionVectorExpected"
    Write-Host "production_vector_actual_sha256=$productionVectorActual"
    if ($productionVectorExpected -ne $productionVectorActual) { throw "Production vector SHA256 checksum mismatch; refusing to execute vector gate" }
    Write-Host "production-vector-sha256=ok"

    if (Test-Path $MetadataPath) { Write-Host "`n=== Build metadata ==="; Get-Content $MetadataPath }
    if (-not [string]::IsNullOrWhiteSpace($ResolvedEvidenceDirectory)) {
        Copy-Item $ChecksumPath (Join-Path $ResolvedEvidenceDirectory "miner-SHA256SUMS.txt") -Force
        if (Test-Path $MetadataPath) {
            Copy-Item $MetadataPath (Join-Path $ResolvedEvidenceDirectory "miner-build-metadata.txt") -Force
        }
    }
    if (Get-Command nvidia-smi -ErrorAction SilentlyContinue) { Invoke-KhushiStep "NVIDIA driver/GPU report (nvidia-smi)" { nvidia-smi } }
    Invoke-KhushiStep "GPU discovery (--list-devices)" { & $MinerPath --list-devices }
    Invoke-KhushiStep "Canonical hardware vector (--vector-self-test)" { & $MinerPath --device $Device --vector-self-test }
    Invoke-KhushiStep "Production memory/chunk allocation (--production-memory-self-test)" { & $MinerPath --device $Device --production-memory-self-test }
    Write-Host "hardware-production-memory=passed"
    Invoke-KhushiStep "Production dataset boundary vectors" { & $ProductionVectorPath --device $Device }
    Write-Host "hardware-production-vectors=passed"
    Invoke-KhushiStep "GPU benchmark (--benchmark)" { & $MinerPath --device $Device --benchmark $BenchmarkSeconds }
    if (Get-Command nvidia-smi -ErrorAction SilentlyContinue) { Invoke-KhushiStep "GPU telemetry" { & $MinerPath --device $Device --telemetry } }
    Write-Host "`nhardware-vector-memory-and-benchmark=passed"
    Write-Host "This result is evidence for the hardware interoperability gate; it does not activate network mining or consensus."
    Write-Host "Live --mine remains gated; controlled staging uses the isolated staging challenge API instead."

    if ($SubmitStagingSolution) {
        $normalizedEndpoint = $StagingEndpoint.TrimEnd("/")
        $challengeUrl = "$normalizedEndpoint/v1/mining/staging/challenge"
        $submitUrl = "$normalizedEndpoint/v1/mining/staging/submit"
        Write-Host "`n=== Controlled staging GPU solution ==="
        Write-Host "network-submission=explicitly-requested"
        Write-Host "staging_endpoint=$normalizedEndpoint"

        $challenge = Invoke-RestMethod -Method Get -Uri $challengeUrl -TimeoutSec 15
        if ($challenge.algorithm -ne "sudharma-gpupow-v1") { throw "Unexpected staging algorithm: $($challenge.algorithm)" }
        if ($challenge.staging -ne $true) { throw "Endpoint did not return explicit staging work" }
        if ([UInt64]$challenge.height -ne 0) { throw "Current staging hardware gate requires height=0" }
        if ([UInt32]$challenge.cache_nodes -ne 8) { throw "Current staging hardware gate requires cache_nodes=8" }
        if ([string]::IsNullOrWhiteSpace([string]$challenge.challenge_id)) { throw "Staging challenge_id is missing" }
        if ([string]::IsNullOrWhiteSpace([string]$challenge.header_prefix)) { throw "Staging header_prefix is missing" }
        if ([string]::IsNullOrWhiteSpace([string]$challenge.target)) { throw "Staging target is missing" }

        if (-not [string]::IsNullOrWhiteSpace($ResolvedEvidenceDirectory)) {
            $challenge | ConvertTo-Json -Depth 10 | Set-Content -Encoding utf8 (Join-Path $ResolvedEvidenceDirectory "challenge.json")
        }

        Write-Host "challenge_id=$($challenge.challenge_id)"
        Write-Host "staging_height=$($challenge.height)"
        Write-Host "staging_cache_nodes=$($challenge.cache_nodes)"

        $stagingArgs = @(
            "--device", [string]$Device,
            "--staging-search",
            "--header-prefix-hex", [string]$challenge.header_prefix,
            "--target-hex", [string]$challenge.target,
            "--height", [string]$challenge.height,
            "--cache-nodes", [string]$challenge.cache_nodes
        )
        $stagingOutput = @(& $MinerPath @stagingArgs 2>&1)
        $stagingExitCode = $LASTEXITCODE
        $stagingOutput | ForEach-Object { Write-Host $_ }
        if ($stagingExitCode -ne 0) {
            throw "staging GPU search failed with exit code $stagingExitCode"
        }

        $nonceLine = $stagingOutput | Where-Object { [string]$_ -match '^staging-solution-nonce=([0-9]+)' } | Select-Object -First 1
        if (-not $nonceLine) { throw "GPU staging search returned no staging-solution-nonce=" }
        $match = [regex]::Match([string]$nonceLine, '^staging-solution-nonce=([0-9]+)')
        if (-not $match.Success) { throw "Unable to parse staging solution nonce" }
        $nonce = [UInt64]::Parse($match.Groups[1].Value)
        Write-Host "staging_solution_nonce=$nonce"

        $solution = [ordered]@{
            challenge = $challenge
            nonce = $nonce
        }
        if (-not [string]::IsNullOrWhiteSpace($ResolvedEvidenceDirectory)) {
            $solution | ConvertTo-Json -Depth 10 | Set-Content -Encoding utf8 (Join-Path $ResolvedEvidenceDirectory "solution.json")
        }
        $solutionJson = $solution | ConvertTo-Json -Depth 10 -Compress
        $result = Invoke-RestMethod -Method Post -Uri $submitUrl -ContentType "application/json" -Body $solutionJson -TimeoutSec 15
        if (-not [string]::IsNullOrWhiteSpace($ResolvedEvidenceDirectory)) {
            $result | ConvertTo-Json -Depth 10 | Set-Content -Encoding utf8 (Join-Path $ResolvedEvidenceDirectory "submit-result.json")
        }
        if ($result.status -ne "accepted") {
            throw "Independent staging verifier rejected GPU solution with status: $($result.status)"
        }
        Write-Host "staging-submit=accepted"
        Write-Host "network-submission=staging-accepted"
        Write-Host "The GPU nonce was accepted by the isolated independent Go staging verifier. No block was created and consensus was not activated."
    } else {
        Write-Host "network-submission=not-requested"
        Write-Host "Benchmark/self-test mode is the default. Controlled submission requires both -SubmitStagingSolution and -StagingEndpoint."
    }
}
finally {
    Stop-Transcript | Out-Null
    Write-Host "hardware_test_log=$LogPath"
}
