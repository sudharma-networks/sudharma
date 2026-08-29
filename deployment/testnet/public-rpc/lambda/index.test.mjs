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

test('dispatches website visitor requests locally without seed access and allows cross-origin website reads', async () => {
  let upstreamCalls = 0;
  const visitorCalls = [];
  const handler = createHandler({
    seeds,
    fetchImpl: async () => { upstreamCalls += 1; throw new Error('must not proxy visitor route'); },
    visitorHandler: async (request) => {
      visitorCalls.push(request.kind);
      return { statusCode: 200, payload: { total: request.kind === 'websiteVisitorsRecord' ? 12 : 11 } };
    },
    logger: { info() {}, warn() {}, error() {} },
  });

  const read = await handler(event('GET', '/v1/website/visitors'));
  assert.equal(read.statusCode, 200);
  assert.match(read.body, /"total":11/);
  assert.equal(read.headers['access-control-allow-origin'], '*');

  const record = await handler(event('POST', '/v1/website/visitors', JSON.stringify({ visitorId: '11111111-2222-4333-8444-555555555555' })));
  assert.equal(record.statusCode, 200);
  assert.match(record.body, /"total":12/);
  assert.equal(record.headers['access-control-allow-origin'], '*');
  assert.deepEqual(visitorCalls, ['websiteVisitorsRead', 'websiteVisitorsRecord']);
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
