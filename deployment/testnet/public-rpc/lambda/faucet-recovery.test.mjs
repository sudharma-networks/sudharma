import test from 'node:test';
import assert from 'node:assert/strict';
import {
  COIN,
  FaucetError,
  createFaucetService,
  createSigner,
} from './faucet.mjs';

const ADDRESS = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
const CHALLENGE_TX_ID = 'c'.repeat(64);
const NOW = 1_700_000_000_000;

test('prepared initial payout is safely resubmitted with the same transaction id after an uncertain broadcast', async () => {
  const signer = createSigner('0'.repeat(63) + '1');
  const prepared = signer.signTransaction(ADDRESS, 100 * COIN, 7);
  const calls = [];
  const store = {
    async reserveInitial() { return false; },
    async getAddress() {
      return {
        initial_status: 'prepared',
        initial_txid: prepared.ID,
        initial_payout: prepared,
      };
    },
    async markInitialSubmitted(address, txid) { calls.push(['markInitialSubmitted', address, txid]); },
    async completeInitial() { throw new Error('should not complete before confirmation'); },
  };
  const rpc = {
    async transaction(txid) {
      assert.equal(txid, prepared.ID);
      throw new FaucetError(404, 'transaction not found');
    },
    async submit(tx) {
      calls.push(['submit', tx.ID]);
      assert.deepEqual(tx, prepared);
      return { accepted: true, transaction_id: tx.ID };
    },
  };

  const service = createFaucetService({ store, rpc, signer, now: () => NOW });
  const result = await service.requestInitial(ADDRESS);

  assert.equal(result.status, 'submitted');
  assert.equal(result.transaction_id, prepared.ID);
  assert.deepEqual(calls, [
    ['submit', prepared.ID],
    ['markInitialSubmitted', ADDRESS, prepared.ID],
  ]);
});

test('already completed challenge reward is idempotent on retry even during cooldown', async () => {
  const signer = createSigner('0'.repeat(63) + '1');
  const rewardTxId = 'd'.repeat(64);
  const store = {
    async getChallenge(txid) {
      assert.equal(txid, CHALLENGE_TX_ID);
      return {
        address: ADDRESS,
        round: 1,
        status: 'paid',
        reward_txid: rewardTxId,
        completed_at: NOW - 1_000,
      };
    },
    async getAddress() { throw new Error('idempotent retry must return before cooldown state checks'); },
  };
  const rpc = {};

  const service = createFaucetService({ store, rpc, signer, now: () => NOW });
  const result = await service.claimChallenge(ADDRESS, CHALLENGE_TX_ID);

  assert.equal(result.round, 1);
  assert.equal(result.reward_transaction_id, rewardTxId);
  assert.equal(result.status, 'submitted');
});

test('prepared challenge reward is completed from observed pending payout without creating a second reward', async () => {
  const signer = createSigner('0'.repeat(63) + '1');
  const preparedReward = signer.signTransaction(ADDRESS, 50 * COIN, 8);
  const calls = [];
  const store = {
    async getChallenge() {
      return {
        address: ADDRESS,
        round: 1,
        status: 'prepared',
        reward_txid: preparedReward.ID,
        reward_payout: preparedReward,
      };
    },
    async completeChallenge(address, challengeTxId, rewardTxId) {
      calls.push(['completeChallenge', address, challengeTxId, rewardTxId]);
    },
  };
  const rpc = {
    async transaction(txid) {
      assert.equal(txid, preparedReward.ID);
      return { status: 'pending', confirmations: 0 };
    },
    async submit() { throw new Error('must not resubmit an already observed payout'); },
  };

  const service = createFaucetService({ store, rpc, signer, now: () => NOW });
  const result = await service.claimChallenge(ADDRESS, CHALLENGE_TX_ID);

  assert.equal(result.reward_transaction_id, preparedReward.ID);
  assert.equal(result.round, 1);
  assert.deepEqual(calls, [
    ['completeChallenge', ADDRESS, CHALLENGE_TX_ID, preparedReward.ID],
  ]);
});
