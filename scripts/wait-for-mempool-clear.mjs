#!/usr/bin/env node
/**
 * Poll public RPC until mempool is empty (or height advances with lower mempool).
 * Optionally POST /v1/miner/wake on each attempt to nudge demand miners.
 */
const RPC_BASE_URL = process.env.RPC_BASE_URL || 'https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com';
const MAX_ATTEMPTS = Number.parseInt(process.env.MEMPOOL_WAIT_ATTEMPTS || '36', 10);
const SLEEP_MS = Number.parseInt(process.env.MEMPOOL_WAIT_SLEEP_MS || '10000', 10);
const WAKE_MINER = process.env.WAKE_MINER !== 'false';

async function fetchStatus() {
  const response = await fetch(`${RPC_BASE_URL}/v1/status`, { signal: AbortSignal.timeout(8000) });
  if (!response.ok) throw new Error(`status HTTP ${response.status}`);
  return response.json();
}

async function wakeMiner() {
  if (!WAKE_MINER) return;
  try {
    await fetch(`${RPC_BASE_URL}/v1/miner/wake`, { method: 'POST', signal: AbortSignal.timeout(5000) });
  } catch {
    // Best-effort; seed proxy may not expose wake yet.
  }
}

async function main() {
  let before = null;
  for (let attempt = 1; attempt <= MAX_ATTEMPTS; attempt++) {
    await wakeMiner();
    const status = await fetchStatus();
    if (before == null) before = { height: status.height, mempool: status.mempool };
    const cleared = Number(status.mempool || 0) === 0;
    const progressed = status.height > before.height && status.mempool < before.mempool;
    console.log(JSON.stringify({
      attempt,
      height: status.height,
      mempool: status.mempool,
      cleared,
      progressed,
    }));
    if (cleared) {
      console.log(JSON.stringify({ mempool_clear: 'ok', before, after: status }, null, 2));
      return;
    }
    if (attempt < MAX_ATTEMPTS) await new Promise((r) => setTimeout(r, SLEEP_MS));
  }
  const finalStatus = await fetchStatus();
  console.error(JSON.stringify({ mempool_clear: 'timeout', before, after: finalStatus }, null, 2));
  process.exit(1);
}

main().catch((error) => {
  console.error(String(error?.message || error));
  process.exit(1);
});
