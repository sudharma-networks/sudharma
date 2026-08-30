#!/usr/bin/env node
import { analyzeFaucetPublicState } from './analyze-faucet-public-state.mjs';
import { evaluateFaucetRecoveryReadiness } from './evaluate-faucet-recovery-readiness.mjs';

const RPC_BASE_URL = process.env.RPC_BASE_URL || 'https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com';
const FAUCET_SIGNER = process.env.FAUCET_SIGNER || '9ccdc094489874bed888ffe4bdf9b8298f4c5131';
const FAILED_TXID = process.env.FAILED_TXID || 'b66bde192c92d192b47ca6e972911e4bfb310a0f343b6ef15b77c468d8a989cc';

async function fetchJson(path) {
  const response = await fetch(`${RPC_BASE_URL}${path}`, { redirect: 'error' });
  const body = await response.json().catch(() => ({}));
  return { status: response.status, body, ok: response.ok };
}

async function main() {
  const [faucetInfo, faucetHealth, faucetDiagnostics, signerAccount, networkStatus, mempool, failedTx] = await Promise.all([
    fetchJson('/v1/faucet/info'),
    fetchJson('/v1/faucet/health'),
    fetchJson('/v1/faucet/diagnostics'),
    fetchJson(`/v1/accounts/${FAUCET_SIGNER}`),
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
  });

  const readiness = evaluateFaucetRecoveryReadiness(analysis);

  const snapshot = {
    checked_at: new Date().toISOString(),
    height: analysis.network_height ?? null,
    mempool: analysis.network_mempool ?? null,
    likely_blocker: analysis.likely_blocker ?? null,
    chain_advancement_required: analysis.chain_advancement_required ?? false,
    operator_actions: analysis.operator_actions ?? [],
  };

  process.stdout.write(`${JSON.stringify({ snapshot, readiness }, null, 2)}\n`);
  process.exit(readiness.should_attempt_recovery ? 0 : 2);
}

main().catch((error) => {
  process.stderr.write(`${error?.stack || error}\n`);
  process.exit(1);
});
