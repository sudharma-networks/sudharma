#!/usr/bin/env bash
set -euo pipefail

REWARD_ADDRESS="${REWARD_ADDRESS:-9ccdc094489874bed888ffe4bdf9b8298f4c5131}"
MINER_BINARY="${MINER_BINARY:-/usr/local/libexec/sudharma-demand-miner/sudharmad}"
SEED_PEER="${SEED_PEER:-127.0.0.1:28444}"
BLOCKS="${BLOCKS:-1}"

for candidate in \
  "$MINER_BINARY" \
  /usr/local/bin/sudharmad \
  /usr/local/libexec/sudharma-demand-miner/sudharmad \
  "$(command -v sudharmad 2>/dev/null || true)"; do
  if [ -n "$candidate" ] && [ -x "$candidate" ]; then
    MINER_BINARY="$candidate"
    break
  fi
done

if [ -z "$MINER_BINARY" ] || [ ! -x "$MINER_BINARY" ]; then
  echo "faucet_refill_mine: miner binary unavailable" >&2
  exit 1
fi

run_dir="$(mktemp -d /tmp/faucet-refill-XXXXXX)"
cleanup() { rm -rf "$run_dir"; }
trap cleanup EXIT

node_id="faucet-refill-$(date +%s)"
echo "faucet_refill_mine: start blocks=$BLOCKS reward=$REWARD_ADDRESS"

for attempt in $(seq 1 "$BLOCKS"); do
  if ! "$MINER_BINARY" \
    -nodeid "$node_id-$attempt" \
    -listen "127.0.0.1:0" \
    -peer "$SEED_PEER" \
    -datadir "$run_dir/$attempt" \
    -mineblocks 1 \
    -testmineraddress "$REWARD_ADDRESS"; then
    echo "faucet_refill_mine: block $attempt failed" >&2
    exit 1
  fi
  echo "faucet_refill_mine: block $attempt ok"
done

echo "faucet_refill_mine: ok"
