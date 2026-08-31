import test from 'node:test';
import assert from 'node:assert/strict';
import { createOperationTimer, createRuntimeFaucetHandler } from './faucet-runtime.mjs';

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
