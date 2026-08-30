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

test('prepared initial payout is safely resubmitted with the same transaction id under the payout lock after an uncertain broadcast', async () => {
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
    async acquirePayoutLock() { calls.push(['lock']); return true; },
    async releasePayoutLock() { calls.push(['unlock']); },
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
    ['lock'],
    ['submit', prepared.ID],
    ['unlock'],
    ['markInitialSubmitted', ADDRESS, prepared.ID],
  ]);
});

test('prepared payout recovery fails closed when the payout lock is busy', async () => {
  const signer = createSigner('0'.repeat(63) + '1');
  const prepared = signer.signTransaction(ADDRESS, 100 * COIN, 7);
  const store = {
    async reserveInitial() { return false; },
    async getAddress() {
      return { initial_status: 'prepared', initial_txid: prepared.ID, initial_payout: prepared };
    },
    async acquirePayoutLock() { return false; },
    async releasePayoutLock() { throw new Error('must not unlock a lock that was not acquired'); },
  };
  const rpc = {
    async transaction() { throw new Error('must not touch RPC without the payout lock'); },
    async submit() { throw new Error('must not submit without the payout lock'); },
  };

  const service = createFaucetService({ store, rpc, signer, now: () => NOW });
  await assert.rejects(service.requestInitial(ADDRESS), /faucet is busy/i);
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

test('prepared challenge reward is completed from observed pending payout under the payout lock without creating a second reward', async () => {
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
    async acquirePayoutLock() { calls.push(['lock']); return true; },
    async releasePayoutLock() { calls.push(['unlock']); },
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
    ['lock'],
    ['unlock'],
    ['completeChallenge', ADDRESS, CHALLENGE_TX_ID, preparedReward.ID],
  ]);
});
