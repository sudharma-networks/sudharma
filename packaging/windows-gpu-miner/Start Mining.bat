@echo off
setlocal
title Sudharma GPU Miner
cd /d "%~dp0"
echo.
echo  Sudharma GPU Miner
echo  First time: enter your wallet address.
echo  After that: double-click starts mining automatically.
echo.
if exist "%LOCALAPPDATA%\Sudharma\gpu-miner\reward-address.txt" (
  "%~dp0sudharma-miner.exe" --auto
) else (
  "%~dp0sudharma-miner.exe"
)
echo.
pause
