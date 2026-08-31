@echo off
setlocal
title Sudharma GPU Miner
echo.
echo  Sudharma GPU Miner — Khushi Algorithm
echo  GPU only. Not CPU. Not ASIC.
echo  Public-testnet and mainnet mining are GPU-only.
echo.
set /p ADDR=Paste your 40-character Sudharma wallet address: 
if "%ADDR%"=="" (
  echo Address is required.
  pause
  exit /b 1
)
"%~dp0sudharma-miner.exe" --address %ADDR% --network public-testnet
echo.
pause
