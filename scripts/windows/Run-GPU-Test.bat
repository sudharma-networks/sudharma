@echo off
setlocal EnableExtensions
cd /d "%~dp0"
set "LOG=%~dp0khushi-hardware-test-launcher.log"
>"%LOG%" echo Khushi Hardware Test v0.2.1 launcher
>>"%LOG%" echo Protocol: sudharma-gpupow-v1
>>"%LOG%" echo Local verifier: 127.0.0.1:28646 only
>>"%LOG%" echo Started: %DATE% %TIME%

echo ============================================================
echo Khushi Hardware Test v0.2.1
echo Protocol: sudharma-gpupow-v1
echo Local verifier stays on 127.0.0.1:28646 only.
echo No block is created and network mining is not activated.
echo Launcher log: %LOG%
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
echo a full 60-second selected-profile benchmark, and localhost verification.
echo This can take a little while; this window will remain open when finished.
echo.
>>"%LOG%" echo Hardware gate started: %DATE% %TIME%

powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%BUNDLE%\run-local-staging-gate.ps1" ^
  -VerifierPath "%BUNDLE%\sudharma-gpupow-staging.exe" ^
  -MinerPath "%MINER%" ^
  -Device %device% ^
  -BenchmarkSeconds 60

set "RC=%ERRORLEVEL%"
>>"%LOG%" echo Hardware gate exit code: %RC%
if not "%RC%"=="0" set "FAIL_REASON=hardware/local-staging gate failed with exit code %RC%"

:finish
echo.
if "%RC%"=="" set "RC=0"
if "%RC%"=="0" (
  echo Hardware test process completed.
  echo Confirm the transcript contains: local-staging-gate=accepted
  echo Keep the generated khushi-staging-evidence-* directory for review.
  >>"%LOG%" echo Result: completed; inspect transcript for local-staging-gate=accepted
) else (
  echo Hardware test failed: %FAIL_REASON%
  echo Exit code: %RC%
  >>"%LOG%" echo Result: FAILED - %FAIL_REASON%
)
echo Launcher log: %LOG%
>>"%LOG%" echo Finished: %DATE% %TIME%
echo.
echo Press any key to close this window.
pause >nul
exit /b %RC%
