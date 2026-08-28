param(
    [string]$VerifierPath = ".\sudharma-gpupow-staging.exe",
    [string]$MinerPath = ".\khushi-miner-nvidia.exe",
    [int]$Device = 0,
    [int]$BenchmarkSeconds = 60
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$VerifierPath = (Resolve-Path $VerifierPath).Path
$MinerPath = (Resolve-Path $MinerPath).Path
$BundleDir = Split-Path -Parent $VerifierPath
$VerifierName = Split-Path -Leaf $VerifierPath
$ChecksumPath = Join-Path $BundleDir "SHA256SUMS.txt"
$HardwareScript = Join-Path $BundleDir "test-khushi-miner.ps1"
$Endpoint = "http://127.0.0.1:28646"
$VerifierStdout = Join-Path $BundleDir "staging-verifier.stdout.log"
$VerifierStderr = Join-Path $BundleDir "staging-verifier.stderr.log"

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
Write-Host "staging_endpoint=$Endpoint"
Write-Host "staging_binding=localhost-only"
Write-Host "seed-services=untouched"
Write-Host "consensus-activation=disabled"

$verifier = $null
try {
    Remove-Item $VerifierStdout, $VerifierStderr -ErrorAction SilentlyContinue
    $verifier = Start-Process \
        -FilePath $VerifierPath \
        -ArgumentList @("-listen", "127.0.0.1:28646") \
        -PassThru \
        -WindowStyle Hidden \
        -RedirectStandardOutput $VerifierStdout \
        -RedirectStandardError $VerifierStderr

    $ready = $false
    for ($attempt = 1; $attempt -le 30; $attempt++) {
        if ($verifier.HasExited) {
            $stderrText = if (Test-Path $VerifierStderr) { Get-Content $VerifierStderr -Raw } else { "" }
            throw "Local staging verifier exited early with code $($verifier.ExitCode): $stderrText"
        }
        try {
            $challenge = Invoke-RestMethod -Method Get -Uri "$Endpoint/v1/mining/staging/challenge" -TimeoutSec 2
            if ($challenge.staging -eq $true -and $challenge.algorithm -eq "sudharma-gpupow-v1") {
                $ready = $true
                break
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

    & $HardwareScript \
        -MinerPath $MinerPath \
        -Device $Device \
        -BenchmarkSeconds $BenchmarkSeconds \
        -SubmitStagingSolution \
        -StagingEndpoint $Endpoint
    if ($LASTEXITCODE -ne 0) {
        throw "Local hardware staging gate failed with exit code $LASTEXITCODE"
    }

    Write-Host "local-staging-gate=accepted"
    Write-Host "The physical GPU solution was accepted by the independent local Go verifier. No block was created."
}
finally {
    if ($null -ne $verifier -and -not $verifier.HasExited) {
        Stop-Process -Id $verifier.Id -Force
        $verifier.WaitForExit()
    }
    Write-Host "staging_verifier=stopped"
    if (Test-Path $VerifierStdout) { Write-Host "staging_verifier_stdout=$VerifierStdout" }
    if (Test-Path $VerifierStderr) { Write-Host "staging_verifier_stderr=$VerifierStderr" }
}
