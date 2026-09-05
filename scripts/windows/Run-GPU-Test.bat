@echo off
setlocal EnableExtensions
cd /d "%~dp0"
set "LOG=%~dp0khushi-hardware-test-launcher.log"
set "CONSOLE_LOG=%~dp0khushi-hardware-test-console.log"
>"%LOG%" echo Khushi Hardware Test v0.2.2 launcher
>>"%LOG%" echo Compatibility lineage: Khushi Hardware Test v0.2.1
>>"%LOG%" echo Protocol: sudharma-gpupow-v1
>>"%LOG%" echo Local verifier: 127.0.0.1:28646 only
>>"%LOG%" echo Mainnet rehearsal blocks: 50
>>"%LOG%" echo Started: %DATE% %TIME%

echo ============================================================
echo Khushi Hardware Test v0.2.2
echo Compatibility lineage: Khushi Hardware Test v0.2.1
echo Protocol: sudharma-gpupow-v1
echo 50-block isolated mainnet-policy mining rehearsal
echo Public mainnet launch and mining remain OFF.
echo Local verifier stays on 127.0.0.1:28646 only.
echo Launcher log: %LOG%
echo Full console log: %CONSOLE_LOG%
echo ============================================================
echo.
echo [1] NVIDIA CUDA
echo [2] OpenCL ^(AMD / compatible GPU^)
echo.
set /p backend=Select backend [1]: 
if "%backend%"=="" set "backend=1"
set /p device=Select GPU device index [0]: 
if "%device%"=="" set "device=0"

>>"%LOG%" echo Backend selection: %backend%
>>"%LOG%" echo Device selection: %device%

if "%backend%"=="2" goto opencl
if not "%backend%"=="1" (
  set "FAIL_REASON=invalid backend selection %backend%"
  set "RC=64"
  goto finish
)

:nvidia
set "BUNDLE=%~dp0nvidia"
set "MINER=%~dp0nvidia\khushi-miner-nvidia.exe"
goto validate

:opencl
set "BUNDLE=%~dp0opencl"
set "MINER=%~dp0opencl\khushi-miner-opencl.exe"

:validate
if not exist "%MINER%" (
  set "FAIL_REASON=miner not found: %MINER%"
  set "RC=2"
  goto finish
)
if not exist "%BUNDLE%\sudharma-gpupow-staging.exe" (
  set "FAIL_REASON=same-revision staging verifier not found in %BUNDLE%"
  set "RC=2"
  goto finish
)
if not exist "%BUNDLE%\run-local-staging-gate.ps1" (
  set "FAIL_REASON=local staging gate script not found in %BUNDLE%"
  set "RC=2"
  goto finish
)

echo.
echo Running canonical vectors, production memory checks, autotune,
echo 60-second benchmark, then 50 production-consensus Khushi blocks.
echo This window WILL NOT close automatically on success or error.
echo.
>>"%LOG%" echo Hardware/mainnet rehearsal gate started: %DATE% %TIME%

set "KHUSHI_SCRIPT=%BUNDLE%\run-local-staging-gate.ps1"
set "KHUSHI_VERIFIER=%BUNDLE%\sudharma-gpupow-staging.exe"
set "KHUSHI_MINER=%MINER%"
set "KHUSHI_DEVICE=%device%"
set "KHUSHI_CONSOLE_LOG=%CONSOLE_LOG%"

powershell.exe -NoProfile -ExecutionPolicy Bypass -Command ^
  "$ErrorActionPreference='Stop'; try { & $env:KHUSHI_SCRIPT -VerifierPath $env:KHUSHI_VERIFIER -MinerPath $env:KHUSHI_MINER -Device ([int]$env:KHUSHI_DEVICE) -BenchmarkSeconds 60 -RehearsalBlocks 50 *^>^&1 ^| Tee-Object -FilePath $env:KHUSHI_CONSOLE_LOG; exit 0 } catch { $m = ($_ ^| Out-String); $m ^| Tee-Object -FilePath $env:KHUSHI_CONSOLE_LOG -Append ^| Write-Host; exit 1 }"

set "RC=%ERRORLEVEL%"
>>"%LOG%" echo Hardware/mainnet rehearsal exit code: %RC%
if not "%RC%"=="0" set "FAIL_REASON=hardware/mainnet rehearsal failed with exit code %RC%"

:finish
echo.
if "%RC%"=="" set "RC=0"
if "%RC%"=="0" (
  echo Khushi v0.2.2 test process completed.
  echo Confirm the transcript contains: mainnet-rehearsal=accepted
  echo Confirm accepted_blocks=50 and keep the evidence directory for review.
  >>"%LOG%" echo Result: completed; inspect transcript for mainnet-rehearsal=accepted accepted_blocks=50
) else (
  echo Khushi v0.2.2 test FAILED: %FAIL_REASON%
  echo Exit code: %RC%
  echo Copy the error above or send the console log for debugging.
  >>"%LOG%" echo Result: FAILED - %FAIL_REASON%
)
echo Launcher log: %LOG%
echo Full console log: %CONSOLE_LOG%
>>"%LOG%" echo Finished: %DATE% %TIME%
echo.
echo Press any key to close this window.
pause >nul
exit /b %RC%