import test from 'node:test';
import assert from 'node:assert/strict';
import { analyzeFaucetPublicState } from './analyze-faucet-public-state.mjs';

test('public analyzer flags submit rejection when failed tx is absent on chain', () => {
  const analysis = analyzeFaucetPublicState({
    faucetInfo: { enabled: true },
    faucetHealth: { ready: true },
    signerAccount: { balance: 24998000000, confirmed_nonce: 2, next_nonce: 3 },
    networkStatus: { height: 12, mempool: 2 },
    failedTx: { status: 404, body: { error: 'transaction not found' } },
    failedAddressTx: { status: 200, body: {} },
  });

  assert.equal(analysis.failed_payout.txid_matches_nonce_3_params, true);
  assert.equal(analysis.balance_covers_next_grant, true);
  assert.equal(analysis.likely_blocker, 'mempool_contention');
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
