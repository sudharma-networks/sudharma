import test from 'node:test';
import assert from 'node:assert/strict';
import {
  checkFaucetReadiness,
  createOperationTimer,
  createRuntimeFaucetHandler,
} from './faucet-runtime.mjs';

test('dependency timing logs only operation outcome and latency', async () => {
  const records = [];
  const ticks = [100, 137, 200, 241];
  const timer = createOperationTimer({
    logger: { info(value) { records.push(value); } },
    now: () => ticks.shift(),
  });

  assert.equal(await timer('dynamodb.reserve_initial', async () => 'reserved'), 'reserved');
  await assert.rejects(
    timer('seed.account', async () => { throw new Error('private-sensitive-marker'); }),
    /private-sensitive-marker/,
  );

  assert.deepEqual(records, [
    { event: 'faucet_dependency', operation: 'dynamodb.reserve_initial', outcome: 'success', latency_ms: 37 },
    { event: 'faucet_dependency', operation: 'seed.account', outcome: 'error', error_name: 'Error', latency_ms: 41 },
  ]);
  assert.equal(JSON.stringify(records).includes('private-sensitive-marker'), false);
});

test('readiness checks writable state, seed account and payout funding without spending coins', async () => {
  const calls = [];
  const store = {
    async checkReadWrite() { calls.push('store'); },
  };
  const rpc = {
    async account(address) {
      calls.push(`account:${address}`);
      return { balance: 20_000_000_000, next_nonce: 7 };
    },
  };
  const signer = { address: '0123456789abcdef0123456789abcdef01234567' };

  const result = await checkFaucetReadiness({ store, rpc, signer });

  assert.deepEqual(calls, ['store', `account:${signer.address}`]);
  assert.deepEqual(result, { ready: true });
});

test('readiness fails when faucet funding is below one initial grant plus fee', async () => {
  const store = { async checkReadWrite() {} };
  const rpc = { async account() { return { balance: 1, next_nonce: 0 }; } };
  const signer = { address: '0123456789abcdef0123456789abcdef01234567' };

  await assert.rejects(
    checkFaucetReadiness({ store, rpc, signer }),
    /needs funding/i,
  );
});

test('runtime fails closed when AWS faucet configuration is absent', () => {
  assert.throws(
    () => createRuntimeFaucetHandler({
      seeds: ['http://127.0.0.1:1', 'http://127.0.0.1:2'],
      env: {},
    }),
    /AWS configuration is incomplete/,
  );
});
