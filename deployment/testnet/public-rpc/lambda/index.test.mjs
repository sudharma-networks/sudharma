import test from 'node:test';
import assert from 'node:assert/strict';
import { createHandler } from './index.mjs';

const seeds = ['http://172.31.10.171:29100', 'http://172.31.32.195:29100'];

function event(method, rawPath, body = null) {
  return {
    version: '2.0',
    rawPath,
    headers: body == null ? {} : { 'content-type': 'application/json' },
    body,
    isBase64Encoded: false,
    requestContext: { http: { method } },
  };
}

test('returns local 404 for forbidden route without upstream access', async () => {
  let calls = 0;
  const handler = createHandler({
    seeds,
    fetchImpl: async () => { calls++; throw new Error('must not call'); },
    logger: { info() {}, warn() {}, error() {} },
  });
  const result = await handler(event('GET', '/metrics'));
  assert.equal(result.statusCode, 404);
  assert.equal(result.headers['cache-control'], 'no-store');
  assert.equal(calls, 0);
});

test('returns local 405 for wrong method', async () => {
  const handler = createHandler({ seeds, fetchImpl: async () => { throw new Error('must not call'); }, logger: { info() {}, warn() {}, error() {} } });
  const result = await handler(event('POST', '/health'));
  assert.equal(result.statusCode, 405);
});

test('proxies allowed response and emits no-store', async () => {
  const handler = createHandler({
    seeds,
    fetchImpl: async () => new Response('{"height":0,"status":"ready"}', { status: 200, headers: { 'content-type': 'application/json' } }),
    logger: { info() {}, warn() {}, error() {} },
  });
  const result = await handler(event('GET', '/ready'));
  assert.equal(result.statusCode, 200);
  assert.equal(result.headers['cache-control'], 'no-store');
  assert.equal(result.isBase64Encoded, false);
  assert.match(result.body, /"status":"ready"/);
});

test('dispatches faucet request locally instead of proxying it to a seed', async () => {
  let upstreamCalls = 0;
  const faucetCalls = [];
  const handler = createHandler({
    seeds,
    fetchImpl: async () => { upstreamCalls++; throw new Error('must not proxy faucet route'); },
    faucetHandler: async (request) => {
      faucetCalls.push(request.kind);
      return { statusCode: 202, payload: { amount_sudh: 100, status: 'submitted' } };
    },
    logger: { info() {}, warn() {}, error() {} },
  });
  const result = await handler(event('POST', '/v1/faucet/request', JSON.stringify({ address: 'a'.repeat(40) })));
  assert.equal(result.statusCode, 202);
  assert.match(result.body, /"amount_sudh":100/);
  assert.deepEqual(faucetCalls, ['faucetInitial']);
  assert.equal(upstreamCalls, 0);
});

test('returns 503 uncertain response when both seeds are unavailable', async () => {
  const handler = createHandler({
    seeds,
    fetchImpl: async () => { throw new TypeError('network down'); },
    timeoutMs: 20,
    logger: { info() {}, warn() {}, error() {} },
  });
  const body = '{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","signature":"private-fixture-must-not-be-logged"}';
  const result = await handler(event('POST', '/v1/transactions', body));
  assert.equal(result.statusCode, 503);
  assert.equal(result.headers['cache-control'], 'no-store');
  assert.match(result.body, /uncertain|unavailable/i);
  assert.doesNotMatch(result.body, /private-fixture/);
});

test('safe logs never contain signed transaction body', async () => {
  const records = [];
  const logger = {
    info(value) { records.push(JSON.stringify(value)); },
    warn(value) { records.push(JSON.stringify(value)); },
    error(value) { records.push(JSON.stringify(value)); },
  };
  const handler = createHandler({
    seeds,
    fetchImpl: async () => new Response('{"accepted":true,"transaction_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}', { status: 202, headers: { 'content-type': 'application/json' } }),
    logger,
  });
  const secretMarker = 'signed-private-body-marker';
  const body = `{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","signature":"${secretMarker}"}`;
  const result = await handler(event('POST', '/v1/transactions', body));
  assert.equal(result.statusCode, 202);
  assert.equal(records.some((line) => line.includes(secretMarker)), false);
});

test('faucet errors log sanitized upstream category without the seed body', async () => {
  const records = [];
  const handler = createHandler({
    seeds,
    fetchImpl: async () => { throw new Error('must not proxy faucet route'); },
    faucetHandler: async () => {
      const error = new Error('testnet faucet is temporarily unavailable');
      error.statusCode = 503;
      error.upstreamStatus = 422;
      error.errorCategory = 'invalid_signature';
      error.uncertain = true;
      throw error;
    },
    logger: {
      info(value) { records.push(value); },
      warn(value) { records.push(value); },
      error(value) { records.push(value); },
    },
  });
  const result = await handler(event('POST', '/v1/faucet/request', JSON.stringify({ address: 'a'.repeat(40) })));
  assert.equal(result.statusCode, 503);
  const errorLog = records.find((value) => value?.event === 'wallet_faucet_error');
  assert.deepEqual(errorLog, {
    event: 'wallet_faucet_error',
    route: 'faucetInitial',
    status_code: 503,
    http_status: 422,
    error_category: 'invalid_signature',
    latency_ms: errorLog.latency_ms,
    request_id: null,
  });
  assert.equal(JSON.stringify(records).includes('transaction rejected'), false);
});
