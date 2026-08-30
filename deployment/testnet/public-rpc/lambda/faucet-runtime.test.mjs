import test from 'node:test';
import assert from 'node:assert/strict';
import * as runtimeModule from './faucet-runtime.mjs';
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

test('runtime store persists prepared payouts and retrieves challenge claims for recovery', async () => {
  assert.equal(typeof runtimeModule.createStore, 'function', 'runtime store must be testable');

  const commands = [];
  const fakeClient = {
    async send(command) {
      commands.push(command);
      if (command.constructor.name === 'GetCommand' && String(command.input?.Key?.pk || '').startsWith('TX#')) {
        return {
          Item: {
            pk: command.input.Key.pk,
            kind: 'challenge_claim',
            address: '0123456789abcdef0123456789abcdef01234567',
            round: 1,
            status: 'prepared',
          },
        };
      }
      return {};
    },
  };
  const timed = async (_operation, action) => action();
  const store = runtimeModule.createStore('FaucetTable', timed, fakeClient);
  const address = '0123456789abcdef0123456789abcdef01234567';
  const txid = 'a'.repeat(64);
  const payout = { ID: 'b'.repeat(64), From: 'f'.repeat(40), To: address, Amount: 1, Fee: 1, Nonce: 1 };

  await store.prepareInitial(address, payout, 1000);
  await store.prepareChallengePayout(address, txid, payout, 2000);
  const claim = await store.getChallenge(txid);

  assert.equal(claim.status, 'prepared');
  const operations = commands.map((command) => ({ name: command.constructor.name, input: command.input }));
  assert.ok(operations.some(({ name, input }) =>
    name === 'UpdateCommand' &&
    input.Key.pk === `ADDR#${address}` &&
    input.ExpressionAttributeValues[':prepared'] === 'prepared' &&
    input.ExpressionAttributeValues[':payout'].ID === payout.ID,
  ));
  assert.ok(operations.some(({ name, input }) =>
    name === 'UpdateCommand' &&
    input.Key.pk === `TX#${txid}` &&
    input.ExpressionAttributeValues[':prepared'] === 'prepared' &&
    input.ExpressionAttributeValues[':payout'].ID === payout.ID,
  ));
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
