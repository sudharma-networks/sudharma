import test from 'node:test';
import assert from 'node:assert/strict';
import {
  MAX_REQUEST_BYTES,
  matchRoute,
  normalizeEvent,
  validTransactionId,
} from './router.mjs';

const ADDRESS = '0123456789abcdef0123456789abcdef01234567';

test('allows the public wallet and faucet route shapes', () => {
  const cases = [
    ['GET', '/health', 'health'],
    ['GET', '/ready', 'ready'],
    ['GET', '/v1/status', 'status'],
    ['GET', `/v1/accounts/${ADDRESS}`, 'account'],
    ['POST', '/v1/transactions', 'submitTransaction'],
    ['GET', '/v1/transactions/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'transactionStatus'],
    ['GET', '/v1/faucet/health', 'faucetHealth'],
    ['GET', '/v1/faucet/info', 'faucetInfo'],
    ['POST', '/v1/faucet/request', 'faucetInitial'],
    ['POST', '/v1/faucet/challenge', 'faucetChallenge'],
  ];
  for (const [method, path, kind] of cases) {
    assert.equal(matchRoute(method, path).kind, kind, `${method} ${path}`);
  }
});

test('rejects forbidden, malformed and wrong-method routes', () => {
  const cases = [
    ['GET', '/metrics'],
    ['GET', '/v1/blocks/0'],
    ['GET', '/v1/mempool'],
    ['POST', '/health'],
    ['GET', '/v1/accounts/'],
    ['GET', '/v1/accounts/alice'],
    ['GET', '/v1/accounts/0123456789abcdef0123456789abcdef0123456'],
    ['GET', '/v1/accounts/0123456789ABCDEF0123456789ABCDEF01234567'],
    ['GET', '/v1/accounts/a/b'],
    ['GET', '/v1/accounts/%2e%2e%2fmetrics'],
    ['GET', '/v1/transactions/not-a-transaction-id'],
    ['GET', '/v1/transactions/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'],
    ['GET', '/v1/transactions/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA'],
    ['POST', '/v1/faucet/health'],
    ['POST', '/v1/faucet/info'],
    ['GET', '/v1/faucet/request'],
    ['GET', '/v1/faucet/challenge'],
  ];
  for (const [method, path] of cases) {
    assert.throws(() => matchRoute(method, path), undefined, `${method} ${path}`);
  }
});

test('validates lowercase deterministic transaction ids', () => {
  assert.equal(validTransactionId('0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef'), true);
  for (const value of ['', 'abc', 'g123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef', '0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF']) {
    assert.equal(validTransactionId(value), false, value);
  }
});

test('normalizes API Gateway v2 events without changing transaction bytes', () => {
  const body = '{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","signature":"fixture"}';
  const normalized = normalizeEvent({
    rawPath: '/v1/transactions',
    headers: { 'content-type': 'application/json' },
    body,
    isBase64Encoded: false,
    requestContext: { http: { method: 'POST' } },
  });
  assert.equal(normalized.method, 'POST');
  assert.equal(normalized.path, '/v1/transactions');
  assert.deepEqual(normalized.body, Buffer.from(body, 'utf8'));
});

test('normalizes faucet JSON request bodies', () => {
  const body = JSON.stringify({ address: ADDRESS });
  const normalized = normalizeEvent({
    rawPath: '/v1/faucet/request',
    headers: { 'content-type': 'application/json' },
    body,
    isBase64Encoded: false,
    requestContext: { http: { method: 'POST' } },
  });
  assert.equal(normalized.kind, 'faucetInitial');
  assert.deepEqual(normalized.body, Buffer.from(body, 'utf8'));
});

test('normalizes base64 request bodies byte-for-byte', () => {
  const source = Buffer.from([0, 1, 2, 3, 254, 255]);
  const normalized = normalizeEvent({
    rawPath: '/v1/transactions',
    body: source.toString('base64'),
    isBase64Encoded: true,
    requestContext: { http: { method: 'POST' } },
  });
  assert.deepEqual(normalized.body, source);
});

test('rejects oversized request bodies before any upstream call', () => {
  const body = 'x'.repeat(MAX_REQUEST_BYTES + 1);
  assert.throws(() => normalizeEvent({
    rawPath: '/v1/transactions',
    body,
    isBase64Encoded: false,
    requestContext: { http: { method: 'POST' } },
  }), /too large/i);
});
