#!/usr/bin/env node
import { analyzeFaucetPublicState } from './analyze-faucet-public-state.mjs';

const RPC_BASE_URL = process.env.RPC_BASE_URL || 'https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com';
const FAILED_TXID = process.env.FAILED_TXID || 'b66bde192c92d192b47ca6e972911e4bfb310a0f343b6ef15b77c468d8a989cc';

async function fetchJson(path) {
  const response = await fetch(`${RPC_BASE_URL}${path}`, { redirect: 'error' });
  const body = await response.json().catch(() => ({}));
  return { status: response.status, body, ok: response.ok };
}

export function evaluateFaucetRecoveryReadiness(analysis) {
  const chainStatus = analysis?.failed_payout?.chain_status;
  if (chainStatus === 'confirmed' || chainStatus === 'pending') {
    return { should_attempt_recovery: false, reason: 'already_on_chain', chain_status: chainStatus };
  }
  if (analysis?.likely_blocker === 'mempool_nonce_conflict' && (analysis?.network_mempool ?? 0) > 0) {
    return {
      should_attempt_recovery: false,
      reason: 'mempool_nonce_conflict',
      network_mempool: analysis.network_mempool,
      last_error_category: analysis.last_error_category ?? null,
    };
  }
  return {
    should_attempt_recovery: true,
    reason: 'ready_to_retry',
    likely_blocker: analysis?.likely_blocker ?? null,
    network_mempool: analysis?.network_mempool ?? null,
  };
}

async function main() {
  const [faucetInfo, faucetHealth, faucetDiagnostics, signerAccount, networkStatus, mempool, failedTx] = await Promise.all([
    fetchJson('/v1/faucet/info'),
    fetchJson('/v1/faucet/health'),
    fetchJson('/v1/faucet/diagnostics'),
    fetchJson(`/v1/accounts/${process.env.FAUCET_SIGNER || '9ccdc094489874bed888ffe4bdf9b8298f4c5131'}`),
    fetchJson('/v1/status'),
    fetchJson('/v1/mempool?limit=20'),
    fetchJson(`/v1/transactions/${FAILED_TXID}`),
  ]);

  const analysis = analyzeFaucetPublicState({
    faucetInfo: faucetInfo.body,
    faucetHealth: faucetHealth.body,
    faucetDiagnostics: faucetDiagnostics.ok ? faucetDiagnostics.body : null,
    signerAccount: signerAccount.body,
    networkStatus: networkStatus.body,
    mempool,
    failedTx,
    failedAddressTx: { status: 0, body: {} },
    lastErrorCategory: process.env.LAST_ERROR_CATEGORY || undefined,
    lastHttpStatus: process.env.LAST_HTTP_STATUS ? Number(process.env.LAST_HTTP_STATUS) : undefined,
  });

  const readiness = evaluateFaucetRecoveryReadiness(analysis);
  process.stdout.write(`${JSON.stringify({ analysis, readiness }, null, 2)}\n`);
  process.exit(readiness.should_attempt_recovery ? 0 : 2);
}

if (process.argv[1]?.endsWith('evaluate-faucet-recovery-readiness.mjs')) {
  main().catch((error) => {
    process.stderr.write(`${error?.stack || error}\n`);
    process.exit(1);
  });
}
