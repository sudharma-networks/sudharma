#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out_dir="$repo_root/dist/windows-gpu-miner"
zip_path="$repo_root/dist/sudharma-gpu-miner-windows.zip"

mkdir -p "$out_dir"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
  go build -trimpath -o "$out_dir/sudharma-miner.exe" "$repo_root/cmd/sudharma-miner"

cp "$repo_root/packaging/windows-gpu-miner/Start Mining.bat" "$out_dir/"
cp "$repo_root/packaging/windows-gpu-miner/Start Pool Mining.bat" "$out_dir/"
cp "$repo_root/packaging/windows-gpu-miner/README.txt" "$out_dir/"
cp "$repo_root/packaging/windows-gpu-miner/SudharmaMiner.ps1" "$out_dir/"
cp "$repo_root/packaging/windows-gpu-miner/gpu-miner-pool.example.json" "$out_dir/"

(
  cd "$repo_root/dist"
  zip -r sudharma-gpu-miner-windows.zip windows-gpu-miner
  sha256sum sudharma-gpu-miner-windows.zip | tee sudharma-gpu-miner-windows.zip.sha256
)

printf 'Built %s\n' "$zip_path"
