import test from 'node:test';
import assert from 'node:assert/strict';
import { createHandler } from './index.mjs';
import { matchRoute, normalizeEvent } from './router.mjs';
import { proxyWithFailover } from './upstream.mjs';

const seeds = ['http://seed-one.internal', 'http://seed-two.internal'];

function event(method, rawPath, body = null, rawQueryString = '') {
  return {
    version: '2.0',
    rawPath,
    rawQueryString,
    headers: body == null ? {} : { 'content-type': 'application/json' },
    body,
    isBase64Encoded: false,
    requestContext: { http: { method } },
  };
}

test('shared Lambda preserves faucet health, visitor and explorer routes', () => {
  assert.equal(matchRoute('GET', '/v1/faucet/health').kind, 'faucetHealth');
  assert.equal(matchRoute('GET', '/v1/website/visitors').kind, 'websiteVisitorsRead');
  assert.equal(matchRoute('POST', '/v1/website/visitors').kind, 'websiteVisitorsRecord');
  assert.equal(matchRoute('GET', '/v1/explorer/status').kind, 'explorerStatus');
  assert.equal(matchRoute('GET', '/v1/explorer/search').kind, 'explorerSearch');
  assert.equal(matchRoute('GET', '/v1/mining/work').kind, 'miningWorkGet');
  assert.equal(matchRoute('POST', '/v1/mining/submit').kind, 'miningSubmit');
});

test('validated explorer query is preserved and forwarded upstream', async () => {
  const normalized = normalizeEvent(event('GET', '/v1/explorer/search', null, 'q=42'));
  assert.equal(normalized.queryString, 'q=42');
  const calls = [];
  const result = await proxyWithFailover(normalized, {
    seeds,
    timeoutMs: 100,
    fetchImpl: async (url) => {
      calls.push(url);
      return new Response('{"ok":true}', { status: 200, headers: { 'content-type': 'application/json' } });
    },
  });
  assert.equal(result.statusCode, 200);
  assert.deepEqual(calls, ['http://seed-one.internal/v1/explorer/search?q=42']);
});

test('visitor route dispatches locally and explorer responses keep CORS', async () => {
  let upstreamCalls = 0;
  const visitorKinds = [];
  const handler = createHandler({
    seeds,
    visitorHandler: async (request) => {
      visitorKinds.push(request.kind);
      return { statusCode: 200, payload: { total: 7 } };
    },
    fetchImpl: async () => {
      upstreamCalls += 1;
      return new Response('{"network":"sudharma"}', { status: 200, headers: { 'content-type': 'application/json' } });
    },
    logger: { info() {}, warn() {}, error() {} },
  });

  const visitor = await handler(event('GET', '/v1/website/visitors'));
  assert.equal(visitor.statusCode, 200);
  assert.equal(visitor.headers['access-control-allow-origin'], '*');
  assert.deepEqual(visitorKinds, ['websiteVisitorsRead']);
  assert.equal(upstreamCalls, 0);

  const explorer = await handler(event('GET', '/v1/explorer/status'));
  assert.equal(explorer.statusCode, 200);
  assert.equal(explorer.headers['access-control-allow-origin'], '*');
  assert.equal(upstreamCalls, 1);
});

test('faucet responses and OPTIONS preflight include browser CORS headers', async () => {
  const handler = createHandler({
    seeds,
    faucetHandler: async () => ({ statusCode: 200, payload: { ready: true } }),
    fetchImpl: async () => { throw new Error('must not proxy faucet'); },
    logger: { info() {}, warn() {}, error() {} },
  });

  const health = await handler(event('GET', '/v1/faucet/health'));
  assert.equal(health.statusCode, 200);
  assert.equal(health.headers['access-control-allow-origin'], '*');
  assert.equal(health.headers['access-control-allow-methods'], 'GET,POST,OPTIONS');

  const options = await handler(event('OPTIONS', '/v1/faucet/request'));
  assert.equal(options.statusCode, 204);
  assert.equal(options.headers['access-control-allow-origin'], '*');
  assert.equal(options.headers['access-control-allow-headers'], 'content-type');
});

test('faucet health still dispatches locally after shared-route restoration', async () => {
  let upstreamCalls = 0;
  const faucetKinds = [];
  const handler = createHandler({
    seeds,
    faucetHandler: async (request) => {
      faucetKinds.push(request.kind);
      return { statusCode: 200, payload: { ready: true } };
    },
    fetchImpl: async () => { upstreamCalls += 1; throw new Error('must not proxy faucet health'); },
    logger: { info() {}, warn() {}, error() {} },
  });
  const result = await handler(event('GET', '/v1/faucet/health'));
  assert.equal(result.statusCode, 200);
  assert.deepEqual(faucetKinds, ['faucetHealth']);
  assert.equal(upstreamCalls, 0);
});
