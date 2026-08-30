import test from 'node:test';
import assert from 'node:assert/strict';
import { classifyUpstreamError, createOperationTimer, createRpc, createRuntimeFaucetHandler } from './faucet-runtime.mjs';

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

  assert.deepEqual(records.map((value) => JSON.parse(value)), [
    { event: 'faucet_dependency', operation: 'dynamodb.reserve_initial', outcome: 'success', latency_ms: 37 },
    { event: 'faucet_dependency', operation: 'seed.account', outcome: 'error', error_name: 'Error', latency_ms: 41 },
  ]);
  assert.equal(records.every((value) => typeof value === 'string'), true);
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
    async () => {
      try {
        await rpc.submit({ ID: 'a'.repeat(64) });
      } catch (error) {
        assert.equal(error.uncertain, true);
        assert.equal(error.upstreamStatus, 422);
        assert.equal(error.errorCategory, 'invalid_signature');
        throw error;
      }
    },
    /outcome is uncertain/,
  );

  assert.deepEqual(records.map((value) => JSON.parse(value)), [
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

test('RPC diagnostics keep seed rejection bodies out of thrown faucet errors', async () => {
  const timer = createOperationTimer({
    logger: { info() {} },
    now: (() => { let tick = 0; return () => ++tick; })(),
  });
  const rpc = createRpc({
    seeds: ['https://seed-a.example', 'https://seed-b.example'],
    fetchImpl: async () => new Response(
      JSON.stringify({ error: 'transaction rejected by mempool: invalid transaction signature' }),
      { status: 422 },
    ),
    timeoutMs: 100,
    timed: timer,
  });

  await assert.rejects(
    async () => {
      try {
        await rpc.transaction('b'.repeat(64));
      } catch (error) {
        assert.equal(error.upstreamStatus, 422);
        assert.equal(error.errorCategory, 'invalid_signature');
        assert.equal(error.message, 'testnet node rejected request');
        throw error;
      }
    },
    /testnet node rejected request/,
  );
});

test('RPC diagnostics classify additional seed failures from status and safe keywords', async () => {
  const cases = [
    { status: 409, error: 'nonce is stale for this account', category: 'invalid_nonce' },
    { status: 400, error: 'insufficient balance to cover amount and fee', category: 'insufficient_balance' },
    { status: 400, error: 'fee below minimum', category: 'invalid_fee' },
    { status: 409, error: 'transaction already exists', category: 'duplicate_transaction' },
    { status: 400, error: 'invalid transaction id encoding', category: 'invalid_transaction_id' },
    { status: 422, error: 'transaction rejected by policy', category: 'transaction_rejected' },
    { status: 503, error: 'seed overloaded', category: 'upstream_unavailable' },
  ];

  for (const item of cases) {
    const records = [];
    const timer = createOperationTimer({
      logger: { info(value) { records.push(value); } },
      now: (() => { let tick = 0; return () => ++tick; })(),
    });
    const rpc = createRpc({
      seeds: ['https://seed-a.example', 'https://seed-b.example'],
      fetchImpl: async () => new Response(JSON.stringify({ error: item.error }), { status: item.status }),
      timeoutMs: 100,
      timed: timer,
    });

    await assert.rejects(rpc.account('9'.repeat(40)));
    const parsed = JSON.parse(records[0]);
    assert.equal(parsed.http_status, item.status);
    assert.equal(parsed.error_category, item.category);
    assert.equal(JSON.stringify(records).includes(item.error), false);
  }
});

test('upstream classifier maps live seed submit phrases without using raw bodies', () => {
  const cases = [
    { status: 422, error: 'transaction signature or identity is invalid', category: 'invalid_signature' },
    { status: 422, error: 'transaction rejected by mempool validation: transaction rejected by mempool: invalid transaction nonce: expected 3, got 4', category: 'invalid_nonce' },
    { status: 422, error: 'transaction rejected by mempool validation: transaction rejected by mempool: insufficient balance: have 1, need 2', category: 'insufficient_balance' },
    { status: 422, error: 'transaction rejected by mempool validation: transaction rejected by mempool: invalid transaction fee', category: 'invalid_fee' },
    { status: 422, error: 'transaction already exists in mempool: aa', category: 'duplicate_transaction' },
    { status: 422, error: 'transaction already confirmed: aa', category: 'duplicate_transaction' },
    { status: 422, error: 'transaction accepted locally but relay failed: peer timeout', category: 'transaction_rejected' },
    { status: 404, error: 'transaction not found', category: 'not_found' },
  ];
  for (const item of cases) {
    assert.equal(classifyUpstreamError(item.status, item.error), item.category);
  }
});
