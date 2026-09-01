#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

echo "Current mainnet genesis candidate (timestamp may be unset):"
go run ./cmd/sudharma-mainnet-genesis-info | jq .

timestamps=(0 1735689600 1798761600 "$@")
echo
echo "Timestamp previews (engineering only — not an authorized freeze):"
go run ./cmd/sudharma-mainnet-genesis-preview "${timestamps[@]}"

cat <<'EOF'

Set params.MainnetGenesisTimestamp only in a dedicated owner freeze PR.
MainnetLaunchAuthorized must remain false until the final launch decision.
EOF
