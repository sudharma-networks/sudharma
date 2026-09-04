param(
    [string]$VerifierPath = ".\sudharma-gpupow-staging.exe",
    [string]$MinerPath = "",
    [int]$Device = 0,
    [int]$BenchmarkSeconds = 60,
    [int]$RehearsalBlocks = 0
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

if ($RehearsalBlocks -ne 0 -and $RehearsalBlocks -lt 25) {
    throw "RehearsalBlocks must be 0 (legacy compact staging) or at least 25"
}

$VerifierPath = (Resolve-Path $VerifierPath).Path
$BundleDir = Split-Path -Parent $VerifierPath

if ([string]::IsNullOrWhiteSpace($MinerPath)) {
    $packagedMinerCandidates = @(
        @(
            (Join-Path $BundleDir "khushi-miner-nvidia.exe"),
            (Join-Path $BundleDir "khushi-miner-opencl.exe")
        ) | Where-Object { Test-Path $_ }
    )
    if ($packagedMinerCandidates.Count -eq 0) {
        throw "No packaged Khushi miner found beside the staging verifier. Expected khushi-miner-nvidia.exe or khushi-miner-opencl.exe"
    }
    if ($packagedMinerCandidates.Count -gt 1) {
        throw "Multiple packaged Khushi miners found beside the staging verifier; specify -MinerPath explicitly"
    }
    $MinerPath = $packagedMinerCandidates[0]
}

$MinerPath = (Resolve-Path $MinerPath).Path
$VerifierName = Split-Path -Leaf $VerifierPath
$ChecksumPath = Join-Path $BundleDir "SHA256SUMS.txt"
$MetadataPath = Join-Path $BundleDir "build-metadata.txt"
$HardwareScript = Join-Path $BundleDir "test-khushi-miner.ps1"
$Endpoint = "http://127.0.0.1:28646"
$EvidenceDir = Join-Path $BundleDir ("khushi-staging-evidence-{0}" -f (Get-Date -Format "yyyyMMdd-HHmmssfff"))
$EvidenceDir = (New-Item -ItemType Directory -Force -Path $EvidenceDir).FullName
$VerifierStdout = Join-Path $EvidenceDir "staging-verifier.stdout.log"
$VerifierStderr = Join-Path $EvidenceDir "staging-verifier.stderr.log"
$ManifestPath = Join-Path $EvidenceDir "SHA256MANIFEST.txt"

if (-not (Test-Path $ChecksumPath)) {
    throw "SHA256SUMS.txt not found beside verifier: $ChecksumPath"
}
if (-not (Test-Path $HardwareScript)) {
    throw "test-khushi-miner.ps1 not found in staging bundle: $HardwareScript"
}

$checksumLine = Get-Content $ChecksumPath | Where-Object { $_ -match [regex]::Escape($VerifierName) } | Select-Object -First 1
if (-not $checksumLine) {
    throw "No checksum entry for $VerifierName in SHA256SUMS.txt"
}
$expected = (($checksumLine -split '\s+')[0]).ToLowerInvariant()
$actual = (Get-FileHash -Algorithm SHA256 $VerifierPath).Hash.ToLowerInvariant()
Write-Host "staging_verifier_expected_sha256=$expected"
Write-Host "staging_verifier_actual_sha256=$actual"
if ($expected -ne $actual) {
    throw "Staging verifier SHA256 mismatch; refusing to execute"
}
Write-Host "staging_verifier_checksum=ok"
Write-Host "selected_miner=$MinerPath"
Write-Host "staging_endpoint=$Endpoint"
Write-Host "staging_binding=localhost-only"
Write-Host "seed-services=untouched"
Write-Host "public_mainnet_launch=disabled"
Write-Host "public_mainnet_mining=disabled"
Write-Host "evidence_directory=$EvidenceDir"
Write-Host "rehearsal_blocks=$RehearsalBlocks"

$verifier = $null
try {
    $verifierArguments = @("-listen", "127.0.0.1:28646")
    if ($RehearsalBlocks -gt 0) {
        $verifierArguments += @("-mainnet-rehearsal", "-rehearsal-blocks", [string]$RehearsalBlocks)
    }

    $startProcessArgs = @{
        FilePath = $VerifierPath
        ArgumentList = $verifierArguments
        PassThru = $true
        WindowStyle = "Hidden"
        RedirectStandardOutput = $VerifierStdout
        RedirectStandardError = $VerifierStderr
    }
    $verifier = Start-Process @startProcessArgs

    $ready = $false
    for ($attempt = 1; $attempt -le 60; $attempt++) {
        if ($verifier.HasExited) {
            $stderrText = if (Test-Path $VerifierStderr) { Get-Content $VerifierStderr -Raw } else { "" }
            throw "Local staging verifier exited early with code $($verifier.ExitCode): $stderrText"
        }
        try {
            $challenge = Invoke-RestMethod -Method Get -Uri "$Endpoint/v1/mining/staging/challenge" -TimeoutSec 2
            if ($challenge.staging -eq $true -and $challenge.algorithm -eq "sudharma-gpupow-v1") {
                if ($RehearsalBlocks -eq 0 -or ([UInt64]$challenge.height -eq 1 -and [UInt32]$challenge.cache_nodes -eq 262144)) {
                    $ready = $true
                    break
                }
            }
        }
        catch {
            Start-Sleep -Milliseconds 250
        }
    }
    if (-not $ready) {
        throw "Local staging verifier did not become ready at $Endpoint"
    }
    Write-Host "staging_verifier=ready"

    $hardwareArgs = @(
        "-MinerPath", $MinerPath,
        "-Device", $Device,
        "-BenchmarkSeconds", $BenchmarkSeconds,
        "-SubmitStagingSolution",
        "-StagingEndpoint", $Endpoint,
        "-EvidenceDirectory", $EvidenceDir,
        "-RehearsalBlocks", $RehearsalBlocks
    )
    & $HardwareScript @hardwareArgs
    if ($LASTEXITCODE -ne 0) {
        throw "Local hardware staging gate failed with exit code $LASTEXITCODE"
    }

    if ($RehearsalBlocks -gt 0) {
        $status = Invoke-RestMethod -Method Get -Uri "$Endpoint/v1/mining/staging/status" -TimeoutSec 15
        $status | ConvertTo-Json -Depth 10 | Set-Content -Encoding utf8 (Join-Path $EvidenceDir "mainnet-rehearsal-status.json")
        if ($status.completed -ne $true -or [UInt64]$status.accepted_blocks -ne [UInt64]$RehearsalBlocks) {
            throw "Mainnet rehearsal did not complete: accepted=$($status.accepted_blocks) target=$RehearsalBlocks"
        }
        Write-Host "mainnet-rehearsal=accepted accepted_blocks=$($status.accepted_blocks) chain_height=$($status.chain_height)"
        Write-Host "local-staging-gate=accepted"
        Write-Host "All rehearsal blocks were accepted by the production Khushi verifier on an isolated mainnet-policy chain."
        Write-Host "No public mainnet service, seed, RPC or consensus activation was changed."
    } else {
        Write-Host "local-staging-gate=accepted"
        Write-Host "The physical GPU solution was accepted by the compact independent local Go verifier."
    }
}
finally {
    if ($null -ne $verifier -and -not $verifier.HasExited) {
        Stop-Process -Id $verifier.Id -Force
        $verifier.WaitForExit()
    }
    Write-Host "staging_verifier=stopped"

    Copy-Item $ChecksumPath (Join-Path $EvidenceDir "verifier-SHA256SUMS.txt") -Force
    if (Test-Path $MetadataPath) {
        Copy-Item $MetadataPath (Join-Path $EvidenceDir "verifier-build-metadata.txt") -Force
    }

    $blockCreation = if ($RehearsalBlocks -gt 0) { "isolated-mainnet-rehearsal-only" } else { "none" }
    @(
        "gate=khushi-local-staging-interoperability",
        "protocol_id=sudharma-gpupow-v1",
        "endpoint=$Endpoint",
        "verifier_sha256=$actual",
        "rehearsal_blocks=$RehearsalBlocks",
        "public_mainnet_launch=disabled",
        "public_mainnet_mining=disabled",
        "block_creation=$blockCreation",
        "public_chain_submission=none",
        "seed_services=untouched",
        "physical_evidence_gate=not_automatically_completed"
    ) | Set-Content -Encoding ascii (Join-Path $EvidenceDir "gate-metadata.txt")

    $manifestLines = Get-ChildItem -Path $EvidenceDir -File |
        Where-Object { $_.Name -ne "SHA256MANIFEST.txt" } |
        Sort-Object Name |
        ForEach-Object {
            $hash = (Get-FileHash -Algorithm SHA256 $_.FullName).Hash.ToLowerInvariant()
            "$hash  $($_.Name)"
        }
    $manifestLines | Set-Content -Encoding ascii $ManifestPath

    if (Test-Path $VerifierStdout) { Write-Host "staging_verifier_stdout=$VerifierStdout" }
    if (Test-Path $VerifierStderr) { Write-Host "staging_verifier_stderr=$VerifierStderr" }
    Write-Host "evidence_manifest=$ManifestPath"
}
