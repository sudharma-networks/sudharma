import test from 'node:test';
import assert from 'node:assert/strict';
import { attachUpstreamNonceMismatch, checkFaucetDiagnostics, classifyUpstreamError, createOperationTimer, createRpc, createRuntimeFaucetHandler, parsePrometheusGauge } from './faucet-runtime.mjs';

test('dependency timer fails closed when an operation hangs', async () => {
  const timer = createOperationTimer({
    logger: { info() {} },
    timeoutMs: 20,
  });
  const started = Date.now();
  await assert.rejects(
    timer('dynamodb.health_write', () => new Promise(() => {})),
    (error) => error?.statusCode === 503 && /dependency timed out/i.test(error.message),
  );
  assert.ok(Date.now() - started < 500, 'dependency deadline must return before the outer request timeout');
});

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

test('attachUpstreamNonceMismatch extracts expected and submitted nonce safely', () => {
  const error = attachUpstreamNonceMismatch(new Error('seed rejected'), 'invalid transaction nonce: expected 4, got 3');
  assert.equal(error.expectedNonce, 4);
  assert.equal(error.submittedNonce, 3);
});

test('parsePrometheusGauge reads sudharma mempool metric', () => {
  const body = '# HELP sudharma_mempool_transactions Transactions currently in the mempool.\n'
    + '# TYPE sudharma_mempool_transactions gauge\n'
    + 'sudharma_mempool_transactions 2\n';
  assert.equal(parsePrometheusGauge(body, 'sudharma_mempool_transactions'), 2);
});

test('mempool probe fails over to the next seed when the first seed returns a non-json 404', async () => {
  const calls = [];
  const timer = createOperationTimer({ logger: console, now: () => Date.now() });
  const rpc = createRpc({
    seeds: ['https://seed-a.example', 'https://seed-b.example'],
    fetchImpl: async (url) => {
      calls.push(url);
      if (url.startsWith('https://seed-a.example')) {
        return new Response('<html>404</html>', { status: 404, headers: { 'content-type': 'text/html' } });
      }
      return new Response(JSON.stringify({
        count: 1,
        transactions: [{ ID: 'b'.repeat(64), From: '9'.repeat(40), To: 'a'.repeat(40), Nonce: 3, Amount: 1, Fee: 1 }],
      }), { status: 200, headers: { 'content-type': 'application/json' } });
    },
    timeoutMs: 100,
    timed: timer,
  });

  const body = await rpc.mempool(20);
  assert.equal(body.count, 1);
  assert.equal(calls.length, 2);
  assert.match(calls[0], /^https:\/\/seed-a\.example/);
  assert.match(calls[1], /^https:\/\/seed-b\.example/);
});

test('diagnostics include prepared recovery state when recovery address is configured', async () => {
  const recoveryAddress = '16d7dc9ec0495109007860a584c7cf9055da9abf';
  const rpc = {
    async account() {
      return { balance: 24998000000, confirmed_nonce: 2, next_nonce: 3 };
    },
    async status() {
      return { height: 12, mempool: 2 };
    },
    async mempool() {
      throw new Error('mempool unavailable');
    },
    async metrics() {
      throw new Error('metrics unavailable');
    },
  };
  const store = {
    async checkReadWrite() {},
    async getAddress(address) {
      assert.equal(address, recoveryAddress);
      return {
        initial_status: 'prepared',
        initial_txid: 'b'.repeat(64),
        initial_last_error_category: 'invalid_nonce',
        initial_last_http_status: 422,
      };
    },
  };

  const payload = await checkFaucetDiagnostics({
    store,
    rpc,
    signer: { address: '9ccdc094489874bed888ffe4bdf9b8298f4c5131' },
    recoveryAddress,
  });

  assert.equal(payload.prepared_recovery.initial_status, 'prepared');
  assert.equal(payload.prepared_recovery.initial_last_error_category, 'invalid_nonce');
  assert.equal(payload.prepared_recovery.initial_last_http_status, 422);
  assert.equal(payload.mempool_inference.chain_advancement_required, true);
});
