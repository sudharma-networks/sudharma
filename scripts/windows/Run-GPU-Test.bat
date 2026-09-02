@echo off
setlocal EnableExtensions
cd /d "%~dp0"

echo ============================================================
echo Khushi Hardware Test v0.2.0
echo Protocol: sudharma-gpupow-v1
echo Local verifier stays on 127.0.0.1:28646 only.
echo No block is created and network mining is not activated.
echo ============================================================
echo.
echo [1] NVIDIA CUDA
echo [2] OpenCL (AMD / compatible GPU)
echo.
set /p backend=Select backend [1]: 
if "%backend%"=="" set "backend=1"

set /p device=Select GPU device index [0]: 
if "%device%"=="" set "device=0"

if "%backend%"=="2" goto opencl
if not "%backend%"=="1" (
  echo ERROR: invalid backend selection "%backend%".
  exit /b 64
)

:nvidia
set "BUNDLE=%~dp0nvidia"
set "MINER=%~dp0nvidia\khushi-miner-nvidia.exe"
goto run

:opencl
set "BUNDLE=%~dp0opencl"
set "MINER=%~dp0opencl\khushi-miner-opencl.exe"
goto run

:run
if not exist "%MINER%" (
  echo ERROR: miner not found: %MINER%
  exit /b 2
)
if not exist "%BUNDLE%\sudharma-gpupow-staging.exe" (
  echo ERROR: same-revision staging verifier not found in %BUNDLE%.
  exit /b 2
)
if not exist "%BUNDLE%\run-local-staging-gate.ps1" (
  echo ERROR: local staging gate script not found in %BUNDLE%.
  exit /b 2
)

echo.
echo Running selected GPU through canonical vectors, production memory,
echo autotune, full 60-second selected-profile benchmark, and localhost verification.
echo.

powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%BUNDLE%\run-local-staging-gate.ps1" ^
  -VerifierPath "%BUNDLE%\sudharma-gpupow-staging.exe" ^
  -MinerPath "%MINER%" ^
  -Device %device% ^
  -BenchmarkSeconds 60

set "rc=%ERRORLEVEL%"
echo.
if "%rc%"=="0" (
  echo Hardware test process completed.
  echo Confirm the transcript contains: local-staging-gate=accepted
  echo Keep the generated khushi-staging-evidence-* directory for review.
) else (
  echo Hardware test failed with exit code %rc%.
)
exit /b %rc%
