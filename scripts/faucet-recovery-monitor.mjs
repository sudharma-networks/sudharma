#!/usr/bin/env node
import { evaluateFaucetRecoveryReadiness } from './evaluate-faucet-recovery-readiness.mjs';

const RPC_BASE_URL = process.env.RPC_BASE_URL || 'https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com';

async function fetchJson(path) {
  const response = await fetch(`${RPC_BASE_URL}${path}`, { redirect: 'error' });
  const body = await response.json().catch(() => ({}));
  return { status: response.status, body, ok: response.ok };
}

async function main() {
  const [status, diagnostics] = await Promise.all([
    fetchJson('/v1/status'),
    fetchJson('/v1/faucet/diagnostics'),
  ]);

  const snapshot = {
    checked_at: new Date().toISOString(),
    height: status.body?.height ?? null,
    mempool: status.body?.mempool ?? null,
    seed_mempool: diagnostics.ok ? diagnostics.body?.seed_mempool ?? null : null,
    mempool_inference: diagnostics.ok ? diagnostics.body?.mempool_inference ?? null : null,
  };

  const readiness = evaluateFaucetRecoveryReadiness({
    failed_payout: { chain_status: 'not_found' },
    likely_blocker: snapshot.mempool === 0 ? 'submit_rejected_not_on_chain' : 'mempool_nonce_conflict',
    network_mempool: snapshot.mempool,
    last_error_category: snapshot.mempool === 0 ? null : 'invalid_nonce',
  });

  process.stdout.write(`${JSON.stringify({ snapshot, readiness }, null, 2)}\n`);
  process.exit(readiness.should_attempt_recovery ? 0 : 2);
}

main().catch((error) => {
  process.stderr.write(`${error?.stack || error}\n`);
  process.exit(1);
});
