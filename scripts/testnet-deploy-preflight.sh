#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <public-profile.json> <node.json>" >&2
  exit 2
fi

PROFILE="$1"
NODE_CONFIG="$2"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

for file in "$PROFILE" "$NODE_CONFIG"; do
  if [[ ! -f "$file" ]]; then
    echo "missing required file: $file" >&2
    exit 1
  fi
  if grep -q 'REPLACE_WITH_REAL_' "$file"; then
    echo "deployment placeholder remains in $file" >&2
    exit 1
  fi
done

python3 - "$NODE_CONFIG" <<'PY'
import json, sys
path = sys.argv[1]
with open(path, encoding="utf-8") as f:
    cfg = json.load(f)
required = ("node_id", "p2p_address", "rpc_address", "data_directory")
for key in required:
    if not str(cfg.get(key, "")).strip():
        raise SystemExit(f"node configuration missing {key}")
text = json.dumps(cfg).lower()
for forbidden in ("private_key", "privatekey", "wallet_password", "seed_phrase", "mnemonic"):
    if forbidden in text:
        raise SystemExit(f"node configuration must not contain wallet secret field: {forbidden}")
if not bool(cfg.get("metrics", False)):
    raise SystemExit("public testnet node must have metrics enabled")
print(f"node config OK: {cfg['node_id']}")
PY

cd "$ROOT"
go test ./testnet ./operations -count=1
go build ./cmd/sudharma-rpcd ./cmd/sudharma-testnet-manifest

echo "--- public testnet launch manifest ---"
go run ./cmd/sudharma-testnet-manifest -profile "$PROFILE"
echo "deployment preflight OK"
