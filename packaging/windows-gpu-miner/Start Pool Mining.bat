@echo off
setlocal
title Sudharma Pool Miner
cd /d "%~dp0"
echo.
echo  Sudharma Pool Miner (Stratum)
echo  Connect to a Sudharma pool operator over Stratum v1.
echo.
set "POOL_URL="
set /p POOL_URL="Pool URL (stratum+tcp://host:3333): "
if "%POOL_URL%"=="" (
  echo Pool URL is required.
  pause
  exit /b 1
)
if exist "%LOCALAPPDATA%\Sudharma\gpu-miner\reward-address.txt" (
  "%~dp0sudharma-miner.exe" --auto --stratum "%POOL_URL%" --worker rig1
) else (
  "%~dp0sudharma-miner.exe" --stratum "%POOL_URL%" --worker rig1
)
echo.
pause
