import test from 'node:test';
import assert from 'node:assert/strict';
import { analyzeFaucetPublicState } from './analyze-faucet-public-state.mjs';

test('public analyzer flags signer mempool transactions as contention', () => {
  const analysis = analyzeFaucetPublicState({
    faucetInfo: { enabled: false },
    faucetHealth: { ready: true },
    signerAccount: { balance: 24998000000, confirmed_nonce: 2, next_nonce: 3 },
    networkStatus: { height: 12, mempool: 2 },
    mempool: {
      status: 200,
      body: {
        count: 2,
        transactions: [
          { ID: 'a'.repeat(64), From: '9ccdc094489874bed888ffe4bdf9b8298f4c5131', To: 'b'.repeat(40), Nonce: 3, Amount: 10000000000, Fee: 10000000 },
        ],
      },
    },
    failedTx: { status: 404, body: { error: 'transaction not found' } },
    failedAddressTx: { status: 200, body: {} },
  });

  assert.equal(analysis.likely_blocker, 'mempool_contention');
  assert.equal(analysis.signer_mempool_txs.length, 1);
});

test('public analyzer flags submit rejection when failed tx is absent on chain', () => {
  const analysis = analyzeFaucetPublicState({
    faucetInfo: { enabled: true },
    faucetHealth: { ready: true },
    signerAccount: { balance: 24998000000, confirmed_nonce: 2, next_nonce: 3 },
    networkStatus: { height: 12, mempool: 0 },
    mempool: { status: 200, body: { count: 0, transactions: [] } },
    failedTx: { status: 404, body: { error: 'transaction not found' } },
    failedAddressTx: { status: 200, body: {} },
  });

  assert.equal(analysis.failed_payout.txid_matches_nonce_3_params, true);
  assert.equal(analysis.balance_covers_next_grant, true);
  assert.equal(analysis.likely_blocker, 'submit_rejected_not_on_chain');
});

test('public analyzer prioritizes recorded invalid_nonce over generic mempool heuristics', () => {
  const analysis = analyzeFaucetPublicState({
    faucetInfo: { enabled: false },
    faucetHealth: { ready: true },
    signerAccount: { balance: 24998000000, confirmed_nonce: 2, next_nonce: 3 },
    networkStatus: { height: 12, mempool: 2 },
    mempool: { status: 404, body: {}, ok: false },
    failedTx: { status: 404, body: { error: 'transaction not found' } },
    failedAddressTx: { status: 200, body: {} },
    lastErrorCategory: 'invalid_nonce',
    lastHttpStatus: 422,
  });

  assert.equal(analysis.likely_blocker, 'mempool_nonce_conflict');
  assert.equal(analysis.last_error_category, 'invalid_nonce');
});

test('public analyzer infers mempool nonce conflict from diagnostics when seed mempool is unavailable', () => {
  const analysis = analyzeFaucetPublicState({
    faucetDiagnostics: {
      ready: true,
      mempool_inference: { likely_prepared_nonce_blocked: true },
      network: { mempool: 2 },
    },
    networkStatus: { height: 12, mempool: 2 },
    failedTx: { status: 404, body: { error: 'transaction not found' } },
    failedAddressTx: { status: 200, body: {} },
    signerAccount: { balance: 24998000000, confirmed_nonce: 2, next_nonce: 3 },
  });

  assert.equal(analysis.likely_blocker, 'mempool_nonce_conflict');
});

test('public analyzer flags chain advancement required when mempool blocks recovery', () => {
  const analysis = analyzeFaucetPublicState({
    faucetDiagnostics: {
      ready: true,
      mempool_inference: { likely_prepared_nonce_blocked: true, chain_advancement_required: true },
      network: { height: 12, mempool: 2 },
    },
    networkStatus: { height: 12, mempool: 2 },
    failedTx: { status: 404, body: { error: 'transaction not found' } },
    failedAddressTx: { status: 200, body: {} },
    signerAccount: { balance: 24998000000, confirmed_nonce: 2, next_nonce: 3 },
    lastErrorCategory: 'invalid_nonce',
  });

  assert.equal(analysis.chain_advancement_required, true);
  assert.equal(analysis.operator_actions?.length, 3);
});

test('public analyzer reads last error category from diagnostics prepared recovery', () => {
  const analysis = analyzeFaucetPublicState({
    faucetDiagnostics: {
      ready: true,
      prepared_recovery: {
        initial_status: 'prepared',
        initial_last_error_category: 'invalid_nonce',
        initial_last_http_status: 422,
      },
      mempool_inference: { likely_prepared_nonce_blocked: true, chain_advancement_required: true },
      network: { height: 12, mempool: 2 },
    },
    networkStatus: { height: 12, mempool: 2 },
    failedTx: { status: 404, body: { error: 'transaction not found' } },
    failedAddressTx: { status: 200, body: {} },
    signerAccount: { balance: 24998000000, confirmed_nonce: 2, next_nonce: 3 },
  });

  assert.equal(analysis.last_error_category, 'invalid_nonce');
  assert.equal(analysis.last_http_status, 422);
  assert.equal(analysis.likely_blocker, 'mempool_nonce_conflict');
});

test('public analyzer detects insufficient signer balance', () => {
  const analysis = analyzeFaucetPublicState({
    faucetInfo: { enabled: false },
    faucetHealth: { ready: true },
    signerAccount: { balance: 1000, confirmed_nonce: 2, next_nonce: 3 },
    networkStatus: { height: 12, mempool: 0 },
    failedTx: { status: 404, body: {} },
    failedAddressTx: { status: 200, body: {} },
  });

  assert.equal(analysis.likely_blocker, 'insufficient_balance');
});
