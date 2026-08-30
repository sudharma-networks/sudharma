import test from 'node:test';
import assert from 'node:assert/strict';
import { createOperationTimer, createRpc, createRuntimeFaucetHandler } from './faucet-runtime.mjs';

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

test('runtime fails closed when AWS faucet configuration is absent', () => {
  assert.throws(
    () => createRuntimeFaucetHandler({
      seeds: ['http://127.0.0.1:1', 'http://127.0.0.1:2'],
      env: {},
    }),
    /AWS configuration is incomplete/,
  );
});


test('RPC diagnostics classify seed rejection without logging its response body', async () => {
  const records = [];
  const timer = createOperationTimer({
    logger: { info(value) { records.push(value); } },
    now: (() => { let tick = 0; return () => ++tick; })(),
  });
  const responses = [
    new Response(JSON.stringify({ error: 'transaction rejected by mempool: invalid transaction signature' }), { status: 422 }),
    new Response(JSON.stringify({ error: 'transaction not found' }), { status: 404 }),
  ];
  const rpc = createRpc({
    seeds: ['https://seed-a.example', 'https://seed-b.example'],
    fetchImpl: async () => responses.shift(),
    timeoutMs: 100,
    timed: timer,
  });

  await assert.rejects(
    rpc.submit({ ID: 'a'.repeat(64) }),
    /outcome is uncertain/,
  );

  assert.deepEqual(records, [
    {
      event: 'faucet_dependency',
      operation: 'seed.submit_transaction',
      outcome: 'error',
      error_name: 'FaucetError',
      http_status: 422,
      error_category: 'invalid_signature',
      latency_ms: 1,
    },
    {
      event: 'faucet_dependency',
      operation: 'seed.reconcile_transaction',
      outcome: 'error',
      error_name: 'FaucetError',
      http_status: 404,
      error_category: 'not_found',
      latency_ms: 1,
    },
  ]);
  assert.equal(JSON.stringify(records).includes('transaction rejected by mempool'), false);
});
