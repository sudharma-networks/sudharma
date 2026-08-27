import test from 'node:test';
import assert from 'node:assert/strict';
import { createSigner, createFaucetService, COIN } from './faucet.mjs';

const ADDRESS_A = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
const ADDRESS_B = 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb';
const TX_ID = 'c'.repeat(64);

test('private scalar derives the expected Sudharma address and signs a valid transaction', async () => {
  const signer = createSigner('0'.repeat(63) + '1');
  assert.equal(signer.address, '698bea63dc44a344663ff1429aea10842df27b6b');
  const tx = signer.signTransaction(ADDRESS_A, 100 * COIN, 0);
  assert.equal(tx.From, signer.address);
  assert.equal(tx.To, ADDRESS_A);
  assert.equal(tx.Amount, 100 * COIN);
  assert.equal(tx.Fee, 10_000_000);
  assert.equal(tx.Nonce, 0);
  assert.match(tx.ID, /^[0-9a-f]{64}$/);
  assert.equal(Buffer.from(tx.PublicKey, 'base64').length, 65);
  assert.equal(Buffer.from(tx.Signature, 'base64').length, 64);
});

test('initial grant stays submitted until its transaction is confirmed', async () => {
  const calls = [];
  const store = {
    async reserveInitial(address) { calls.push(['reserveInitial', address]); return true; },
    async markInitialSubmitted(address, txid, at) { calls.push(['markInitialSubmitted', address, txid, at]); },
    async completeInitial(address, txid, at) { calls.push(['completeInitial', address, txid, at]); },
    async getAddress() { return null; },
    async acquirePayoutLock() { return true; },
    async releasePayoutLock() {},
  };
  const signer = createSigner('0'.repeat(63) + '1');
  const rpc = {
    async account(address) { return { address, balance: 10_000 * COIN, next_nonce: 7 }; },
    async submit(tx) { calls.push(['submit', tx]); return { accepted: true, transaction_id: tx.ID }; },
  };
  const service = createFaucetService({ store, rpc, signer, now: () => 1_700_000_000_000 });
  const result = await service.requestInitial(ADDRESS_A);
  assert.equal(result.amount_sudh, 100);
  assert.equal(result.status, 'submitted');
  assert.equal(calls.find((x) => x[0] === 'submit')[1].Amount, 100 * COIN);
  assert.ok(calls.some((x) => x[0] === 'markInitialSubmitted'));
  assert.equal(calls.some((x) => x[0] === 'completeInitial'), false);
});

test('repeated initial request reconciles a submitted payout after confirmation', async () => {
  const calls = [];
  const store = {
    async reserveInitial() { return false; },
    async getAddress() {
      return { initial_status: 'submitted', initial_txid: TX_ID };
    },
    async completeInitial(address, txid, at) { calls.push(['completeInitial', address, txid, at]); },
  };
  const rpc = {
    async transaction(txid) {
      assert.equal(txid, TX_ID);
      return { status: 'confirmed', confirmations: 1 };
    },
  };
  const signer = createSigner('0'.repeat(63) + '1');
  const service = createFaucetService({ store, rpc, signer, now: () => 1_700_000_000_000 });
  const result = await service.requestInitial(ADDRESS_A);
  assert.equal(result.status, 'confirmed');
  assert.equal(result.transaction_id, TX_ID);
  assert.ok(calls.some((x) => x[0] === 'completeInitial'));
});

test('challenge requires confirmed exact 25 SUDH payment and returns 50 SUDH', async () => {
  const signer = createSigner('0'.repeat(63) + '1');
  const calls = [];
  const store = {
    async getAddress() { return { initial_status: 'paid', rounds: 0, last_round_at: null }; },
    async reserveChallenge(address, txid, round) { calls.push(['reserveChallenge', address, txid, round]); return true; },
    async completeChallenge(address, txid, payoutTxId, at) { calls.push(['completeChallenge', address, txid, payoutTxId, at]); },
    async acquirePayoutLock() { return true; },
    async releasePayoutLock() {},
  };
  const rpc = {
    async transaction() {
      return {
        status: 'confirmed', confirmations: 1,
        transaction: { From: ADDRESS_A, To: signer.address, Amount: 25 * COIN },
      };
    },
    async account(address) { return { address, balance: 10_000 * COIN, next_nonce: 8 }; },
    async submit(tx) { calls.push(['submit', tx]); return { accepted: true, transaction_id: tx.ID }; },
  };
  const service = createFaucetService({ store, rpc, signer, now: () => 1_700_000_000_000 });
  const result = await service.claimChallenge(ADDRESS_A, TX_ID);
  assert.equal(result.reward_sudh, 50);
  assert.equal(result.round, 1);
  assert.equal(calls.find((x) => x[0] === 'submit')[1].Amount, 50 * COIN);
});

test('challenge rejects early cooldown and sixth round', async () => {
  const signer = createSigner('0'.repeat(63) + '1');
  const baseStore = {
    async reserveChallenge() { throw new Error('should not reserve'); },
    async acquirePayoutLock() { throw new Error('should not lock'); },
  };
  const rpc = { async transaction() { throw new Error('should not query'); } };
  const now = 1_700_000_000_000;

  const cooldownService = createFaucetService({
    store: { ...baseStore, async getAddress() { return { initial_status: 'paid', rounds: 1, last_round_at: now - 60_000 }; } },
    rpc, signer, now: () => now,
  });
  await assert.rejects(() => cooldownService.claimChallenge(ADDRESS_A, TX_ID), /24-hour cooldown/);

  const maxService = createFaucetService({
    store: { ...baseStore, async getAddress() { return { initial_status: 'paid', rounds: 5, last_round_at: now - 48 * 60 * 60 * 1000 }; } },
    rpc, signer, now: () => now,
  });
  await assert.rejects(() => maxService.claimChallenge(ADDRESS_B, TX_ID), /maximum 5 challenge rounds/);
});
