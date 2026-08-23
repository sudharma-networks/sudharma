#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="$ROOT/data-testnet-rehearsal/logs"
mkdir -p "$LOG_DIR"

NODE1_LOG="$LOG_DIR/node1.log"
NODE2_LOG="$LOG_DIR/node2.log"
NODE1_PID=""
NODE2_PID=""

cleanup() {
  set +e
  [[ -n "$NODE2_PID" ]] && kill "$NODE2_PID" 2>/dev/null
  [[ -n "$NODE1_PID" ]] && kill "$NODE1_PID" 2>/dev/null
  [[ -n "$NODE2_PID" ]] && wait "$NODE2_PID" 2>/dev/null
  [[ -n "$NODE1_PID" ]] && wait "$NODE1_PID" 2>/dev/null
}
trap cleanup EXIT INT TERM

wait_ready() {
  local url="$1"
  local name="$2"
  for _ in $(seq 1 40); do
    if curl --fail --silent --max-time 1 "$url/ready" >/dev/null; then
      return 0
    fi
    sleep 0.25
  done
  echo "$name did not become ready" >&2
  return 1
}

cd "$ROOT"
rm -rf data-testnet-rehearsal
mkdir -p "$LOG_DIR"

go run ./cmd/sudharma-rpcd -config testnet/rehearsal/node1.json >"$NODE1_LOG" 2>&1 &
NODE1_PID=$!
wait_ready "http://127.0.0.1:28545" "node1"

go run ./cmd/sudharma-rpcd -config testnet/rehearsal/node2.json >"$NODE2_LOG" 2>&1 &
NODE2_PID=$!
wait_ready "http://127.0.0.1:28546" "node2"

STATUS1="$(curl --fail --silent http://127.0.0.1:28545/v1/status)"
STATUS2="$(curl --fail --silent http://127.0.0.1:28546/v1/status)"

python3 - "$STATUS1" "$STATUS2" <<'PY'
import json, sys
one = json.loads(sys.argv[1])
two = json.loads(sys.argv[2])
for key in ("height", "tip_hash", "total_work"):
    if one.get(key) != two.get(key):
        raise SystemExit(f"rehearsal mismatch for {key}: node1={one.get(key)} node2={two.get(key)}")
if two.get("peers", 0) < 1:
    raise SystemExit("node2 did not retain a P2P peer")
print(f"Sudharma testnet rehearsal OK: height={one['height']} tip={one['tip_hash']} peers(node2)={two['peers']}")
PY
