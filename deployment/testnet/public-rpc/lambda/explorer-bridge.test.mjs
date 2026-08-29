import test from 'node:test';
import assert from 'node:assert/strict';
import { createHandler } from './index.mjs';
import { matchRoute, normalizeEvent } from './router.mjs';
import { proxyWithFailover } from './upstream.mjs';

const ADDRESS = '0123456789abcdef0123456789abcdef01234567';
const HASH = 'a'.repeat(64);
const TXID = 'b'.repeat(64);
const seeds = ['http://seed-one.internal', 'http://seed-two.internal'];

function gatewayEvent(method, rawPath, rawQueryString = '') {
  return {
    version: '2.0',
    rawPath,
    rawQueryString,
    headers: {},
    body: null,
    isBase64Encoded: false,
    requestContext: { http: { method } },
  };
}

test('allows only the documented read-only explorer route shapes', () => {
  const cases = [
    ['GET', '/v1/explorer/status', 'explorerStatus'],
    ['GET', '/v1/explorer/blocks', 'explorerBlocks'],
    ['GET', '/v1/explorer/blocks/42', 'explorerBlock'],
    ['GET', `/v1/explorer/blocks/${HASH}`, 'explorerBlock'],
    ['GET', '/v1/explorer/transactions', 'explorerTransactions'],
    ['GET', `/v1/explorer/transactions/${TXID}`, 'explorerTransaction'],
    ['GET', `/v1/explorer/addresses/${ADDRESS}`, 'explorerAddress'],
    ['GET', '/v1/explorer/search', 'explorerSearch'],
  ];
  for (const [method, path, kind] of cases) {
    assert.equal(matchRoute(method, path).kind, kind, `${method} ${path}`);
  }
});

test('rejects explorer mutations, malformed identifiers and undeclared explorer paths', () => {
  const cases = [
    ['POST', '/v1/explorer/status'],
    ['POST', '/v1/explorer/transactions'],
    ['GET', '/v1/explorer/mempool'],
    ['GET', '/v1/explorer/blocks/not-a-block'],
    ['GET', `/v1/explorer/blocks/${HASH.toUpperCase()}`],
    ['GET', '/v1/explorer/transactions/not-a-transaction'],
    ['GET', `/v1/explorer/addresses/${ADDRESS.toUpperCase()}`],
    ['GET', '/v1/explorer/admin'],
  ];
  for (const [method, path] of cases) {
    assert.throws(() => matchRoute(method, path), undefined, `${method} ${path}`);
  }
});

test('normalizes only documented explorer query parameters', () => {
  const search = normalizeEvent({
    rawPath: '/v1/explorer/search',
    rawQueryString: 'q=42',
    requestContext: { http: { method: 'GET' } },
  });
  assert.equal(search.queryString, 'q=42');

  const blocks = normalizeEvent({
    rawPath: '/v1/explorer/blocks',
    rawQueryString: 'limit=8&before=42',
    requestContext: { http: { method: 'GET' } },
  });
  assert.equal(blocks.queryString, 'limit=8&before=42');

  const transactions = normalizeEvent({
    rawPath: '/v1/explorer/transactions',
    rawQueryString: 'limit=20&cursor=abc_DEF-123',
    requestContext: { http: { method: 'GET' } },
  });
  assert.equal(transactions.queryString, 'limit=20&cursor=abc_DEF-123');

  const address = normalizeEvent({
    rawPath: `/v1/explorer/addresses/${ADDRESS}`,
    rawQueryString: 'limit=20&before_height=100',
    requestContext: { http: { method: 'GET' } },
  });
  assert.equal(address.queryString, 'limit=20&before_height=100');
});

test('rejects unknown, duplicate and out-of-range explorer query values', () => {
  const cases = [
    ['/v1/explorer/status', 'limit=1'],
    ['/v1/explorer/blocks', 'limit=101'],
    ['/v1/explorer/blocks', 'limit=8&limit=9'],
    ['/v1/explorer/transactions', 'admin=1'],
    ['/v1/explorer/transactions', 'cursor=bad%2Fcursor'],
    [`/v1/explorer/addresses/${ADDRESS}`, 'cursor=x&before_height=10'],
    ['/v1/explorer/search', 'q=not-a-chain-identifier'],
  ];
  for (const [rawPath, rawQueryString] of cases) {
    assert.throws(() => normalizeEvent({
      rawPath,
      rawQueryString,
      requestContext: { http: { method: 'GET' } },
    }), undefined, `${rawPath}?${rawQueryString}`);
  }
});

test('forwards the validated explorer query string to the private seed upstream', async () => {
  const calls = [];
  const fetchImpl = async (url) => {
    calls.push(url);
    return new Response('{"type":"block","path":"/explorer/block?id=42"}', {
      status: 200,
      headers: { 'content-type': 'application/json' },
    });
  };
  const result = await proxyWithFailover({
    method: 'GET',
    path: '/v1/explorer/search',
    queryString: 'q=42',
    body: Buffer.alloc(0),
    headers: {},
  }, { seeds, fetchImpl, timeoutMs: 100 });
  assert.equal(result.statusCode, 200);
  assert.equal(calls.length, 1);
  assert.equal(calls[0], 'http://seed-one.internal/v1/explorer/search?q=42');
});

test('adds browser CORS to public explorer responses', async () => {
  const handler = createHandler({
    seeds,
    fetchImpl: async () => new Response('{"network":"sudharma","height":42}', {
      status: 200,
      headers: { 'content-type': 'application/json' },
    }),
    logger: { info() {}, warn() {}, error() {} },
  });
  const result = await handler(gatewayEvent('GET', '/v1/explorer/status'));
  assert.equal(result.statusCode, 200);
  assert.equal(result.headers['access-control-allow-origin'], '*');
});
