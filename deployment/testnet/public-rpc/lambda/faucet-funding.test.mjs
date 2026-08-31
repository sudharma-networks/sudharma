import test from 'node:test';
import assert from 'node:assert/strict';

import { COIN } from './faucet.mjs';
import {
  INITIAL_GRANT_REQUIRED_BALANCE,
  FaucetFundingError,
  requiredPayoutBalance,
  waitForFaucetFunding,
} from './faucet-funding.mjs';

test('requiredPayoutBalance includes the development fee', () => {
  const amount = 100 * COIN;
  assert.equal(requiredPayoutBalance(amount), INITIAL_GRANT_REQUIRED_BALANCE);
  assert.equal(INITIAL_GRANT_REQUIRED_BALANCE, 10_010_000_000);
});

test('waitForFaucetFunding returns once the signer balance is sufficient', async () => {
  const balances = [25 * COIN, 75 * COIN, INITIAL_GRANT_REQUIRED_BALANCE];
  const rpc = {
    account: async () => ({ balance: balances.shift(), next_nonce: 0 }),
  };

  const result = await waitForFaucetFunding({
    rpc,
    signer: { address: 'a'.repeat(40) },
    pollMs: 1,
    sleep: async () => {},
    now: (() => {
      let tick = 0;
      return () => {
        tick += 1;
        return tick;
      };
    })(),
  });

  assert.equal(result.funded, true);
  assert.equal(result.balance, INITIAL_GRANT_REQUIRED_BALANCE);
});

test('waitForFaucetFunding wakes the demand miner while waiting', async () => {
  const wakeCalls = [];
  const rpc = {
    account: async () => ({ balance: 0, next_nonce: 0 }),
    wake: async () => {
      wakeCalls.push(Date.now());
      return { awoken: true };
    },
  };

  await assert.rejects(
    () => waitForFaucetFunding({
      rpc,
      signer: { address: 'a'.repeat(40) },
      timeoutMs: 12,
      pollMs: 5,
      sleep: async () => {},
      wakeMiner: () => rpc.wake(),
      now: (() => {
        let tick = 0;
        return () => {
          tick += 5;
          return tick;
        };
      })(),
    }),
    (error) => error instanceof FaucetFundingError,
  );

  assert.ok(wakeCalls.length >= 2);
});

test('waitForFaucetFunding times out with a funding message', async () => {
  const rpc = {
    account: async () => ({ balance: 0, next_nonce: 0 }),
  };
  let nowMs = 0;
  await assert.rejects(
    () => waitForFaucetFunding({
      rpc,
      signer: { address: 'a'.repeat(40) },
      timeoutMs: 10,
      pollMs: 5,
      sleep: async () => {},
      now: () => {
        nowMs += 5;
        return nowMs;
      },
    }),
    (error) => error instanceof FaucetFundingError
      && error.statusCode === 503
      && /funding/i.test(error.message),
  );
});
