import test from 'node:test';
import assert from 'node:assert/strict';
import {
  COIN,
  FaucetError,
  createFaucetService,
  createSigner,
} from './faucet.mjs';

const ADDRESS = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
const NOW = 1_700_000_000_000;

test('stale prepared initial payout is resubmitted at the current nonce after chain advances', async () => {
  const signer = createSigner('0'.repeat(63) + '1');
  const stalePrepared = signer.signTransaction(ADDRESS, 100 * COIN, 3);
  const calls = [];
  let reserveCalls = 0;
  const store = {
    async reserveInitial(address) {
      reserveCalls += 1;
      calls.push(['reserveInitial', address, reserveCalls]);
      return reserveCalls > 1;
    },
    async getAddress() {
      return {
        initial_status: 'prepared',
        initial_txid: stalePrepared.ID,
        initial_payout: stalePrepared,
      };
    },
    async failInitial(address, message) { calls.push(['failInitial', address, message]); },
    async prepareInitial(address, payout) { calls.push(['prepareInitial', address, payout.ID, payout.Nonce]); },
    async acquirePayoutLock() { calls.push(['lock']); return true; },
    async releasePayoutLock() { calls.push(['unlock']); },
    async markInitialSubmitted(address, txid) { calls.push(['markInitialSubmitted', address, txid]); },
    async completeInitial() { throw new Error('should not complete stale prepared tx'); },
  };
  const rpc = {
    async account() { return { balance: 10_000 * COIN, next_nonce: 4 }; },
    async transaction(txid) {
      assert.equal(txid, stalePrepared.ID);
      throw new FaucetError(404, 'transaction not found');
    },
    async submit(tx) {
      calls.push(['submit', tx.ID, tx.Nonce]);
      assert.equal(tx.Nonce, 4);
      return { accepted: true, transaction_id: tx.ID };
    },
  };

  const service = createFaucetService({ store, rpc, signer, now: () => NOW });
  const result = await service.requestInitial(ADDRESS);

  assert.equal(result.status, 'submitted');
  assert.notEqual(result.transaction_id, stalePrepared.ID);
  assert.ok(calls.some((entry) => entry[0] === 'failInitial'));
  assert.ok(calls.some((entry) => entry[0] === 'submit' && entry[2] === 4));
});
