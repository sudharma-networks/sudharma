#!/usr/bin/env node
import { createHash } from 'node:crypto';

const COIN = 100_000_000;
const INITIAL_GRANT_SUDH = 100;
const RPC_BASE_URL = process.env.RPC_BASE_URL || 'https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com';
const FAUCET_SIGNER = process.env.FAUCET_SIGNER || '9ccdc094489874bed888ffe4bdf9b8298f4c5131';
const FAILED_ADDRESS = process.env.FAILED_ADDRESS || '16d7dc9ec0495109007860a584c7cf9055da9abf';
const FAILED_TXID = process.env.FAILED_TXID || 'b66bde192c92d192b47ca6e972911e4bfb310a0f343b6ef15b77c468d8a989cc';

function grantCostCoin() {
  const amount = INITIAL_GRANT_SUDH * COIN;
  const fee = Math.floor((amount * 10) / 10_000);
  return { amount, fee, total: amount + fee };
}

function expectedTxId(from, to, nonce) {
  const { amount, fee } = grantCostCoin();
  return createHash('sha256')
    .update(`${from}|${to}|${amount}|${fee}|${nonce}`, 'utf8')
    .digest('hex');
}

async function fetchJson(path) {
  const response = await fetch(`${RPC_BASE_URL}${path}`, { redirect: 'error' });
  const body = await response.json().catch(() => ({}));
  return { status: response.status, body, ok: response.ok };
}

export function analyzeFaucetPublicState({
  faucetInfo,
  faucetHealth,
  signerAccount,
  networkStatus,
  mempool,
  failedTx,
  failedAddressTx,
  lastErrorCategory,
  lastHttpStatus,
} = {}) {
  const grant = grantCostCoin();
  const nextNonce = signerAccount?.next_nonce ?? signerAccount?.nextNonce;
  const balance = signerAccount?.balance;
  const expectedFailedTxId = expectedTxId(FAUCET_SIGNER, FAILED_ADDRESS, 3);
  const mempoolTxs = Array.isArray(mempool?.body?.transactions) ? mempool.body.transactions : [];
  const signerMempoolTxs = mempoolTxs
    .filter((tx) => tx?.From === FAUCET_SIGNER)
    .map((tx) => ({ id: tx.ID, to: tx.To, nonce: tx.Nonce, amount: tx.Amount, fee: tx.Fee }));

  const analysis = {
    rpc_base_url: RPC_BASE_URL,
    faucet_enabled: faucetInfo?.enabled ?? null,
    faucet_ready: faucetHealth?.ready ?? null,
    network_height: networkStatus?.height ?? null,
    network_mempool: networkStatus?.mempool ?? null,
    mempool_route_available: mempool?.ok === true,
    signer_mempool_txs: signerMempoolTxs,
    signer: {
      address: FAUCET_SIGNER,
      balance,
      confirmed_nonce: signerAccount?.confirmed_nonce ?? signerAccount?.confirmedNonce ?? null,
      next_nonce: nextNonce,
    },
    failed_payout: {
      address: FAILED_ADDRESS,
      txid: FAILED_TXID,
      expected_txid_for_nonce_3: expectedFailedTxId,
      txid_matches_nonce_3_params: FAILED_TXID === expectedFailedTxId,
      chain_status: failedTx.status === 404 ? 'not_found' : (failedTx.body?.status || 'unknown'),
    },
    grant_cost_coin: grant,
    balance_covers_next_grant: Number.isSafeInteger(balance) && balance >= grant.total,
    mempool_may_block_nonce: signerMempoolTxs.length > 0 || (
      Number.isInteger(networkStatus?.mempool) && networkStatus.mempool > 0 && nextNonce === 3
    ),
  };

  if (typeof lastErrorCategory === 'string' && lastErrorCategory.length > 0) {
    analysis.last_error_category = lastErrorCategory;
  }
  if (Number.isInteger(lastHttpStatus)) {
    analysis.last_http_status = lastHttpStatus;
  }

  if (analysis.last_error_category === 'invalid_nonce') {
    analysis.likely_blocker = 'mempool_nonce_conflict';
    analysis.recommendation = 'Seed mempool already advanced past the prepared nonce; mine blocks or clear conflicting mempool transactions, then resubmit once.';
  } else if (!analysis.balance_covers_next_grant) {
    analysis.likely_blocker = 'insufficient_balance';
    analysis.recommendation = 'Fund the faucet signer before resubmitting nonce-3 payouts.';
  } else if (signerMempoolTxs.some((tx) => tx.id === FAILED_TXID)) {
    analysis.likely_blocker = 'prepared_tx_already_pending';
    analysis.recommendation = 'Reconcile DynamoDB to submitted; the prepared tx is already in mempool.';
  } else if (signerMempoolTxs.length > 0) {
    analysis.likely_blocker = 'mempool_contention';
    analysis.recommendation = 'Clear or mine conflicting faucet mempool transactions before resubmitting nonce 3.';
  } else if (analysis.mempool_may_block_nonce) {
    analysis.likely_blocker = 'mempool_contention';
    analysis.recommendation = 'Inspect seed mempool for stuck transactions before resubmitting nonce 3.';
  } else if (failedTx.status === 404) {
    analysis.likely_blocker = 'submit_rejected_not_on_chain';
    analysis.recommendation = 'Deploy diagnostics-only Lambda, capture seed.submit_transaction error_category, then resubmit prepared payout once.';
  } else if (failedTx.body?.status === 'pending') {
    analysis.likely_blocker = 'already_pending';
    analysis.recommendation = 'Reconcile DynamoDB to submitted; do not resubmit a second copy.';
  } else if (failedTx.body?.status === 'confirmed') {
    analysis.likely_blocker = 'already_confirmed';
    analysis.recommendation = 'Reconcile DynamoDB to paid; faucet recovery is complete for this address.';
  } else {
    analysis.likely_blocker = 'unknown';
    analysis.recommendation = 'Deploy diagnostics-only Lambda and run live diagnostics for error_category.';
  }

  if (failedAddressTx.status === 200 && failedAddressTx.body?.initial_status) {
    analysis.failed_address_state = failedAddressTx.body;
  }

  return analysis;
}

async function main() {
  const [faucetInfo, faucetHealth, signerAccount, networkStatus, mempool, failedTx, failedAddressTx] = await Promise.all([
    fetchJson('/v1/faucet/info'),
    fetchJson('/v1/faucet/health'),
    fetchJson(`/v1/accounts/${FAUCET_SIGNER}`),
    fetchJson('/v1/status'),
    fetchJson('/v1/mempool?limit=20'),
    fetchJson(`/v1/transactions/${FAILED_TXID}`),
    fetchJson(`/v1/explorer/addresses/${FAILED_ADDRESS}?limit=1`).catch(() => ({ status: 0, body: {} })),
  ]);

  const analysis = analyzeFaucetPublicState({
    faucetInfo: faucetInfo.body,
    faucetHealth: faucetHealth.body,
    signerAccount: signerAccount.body,
    networkStatus: networkStatus.body,
    mempool,
    failedTx,
    failedAddressTx,
  });

  process.stdout.write(`${JSON.stringify(analysis, null, 2)}\n`);
}

if (process.argv[1]?.endsWith('analyze-faucet-public-state.mjs')) {
  main().catch((error) => {
    process.stderr.write(`${error?.stack || error}\n`);
    process.exit(1);
  });
}
