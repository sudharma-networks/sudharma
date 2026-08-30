import test from 'node:test';
import assert from 'node:assert/strict';
import { normalizeEvent } from './router.mjs';

const ADDRESS = '0123456789abcdef0123456789abcdef01234567';

test('faucet mutation requests preserve API Gateway source IP for throttling', () => {
  const normalized = normalizeEvent({
    rawPath: '/v1/faucet/request',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ address: ADDRESS }),
    requestContext: {
      http: {
        method: 'POST',
        sourceIp: '203.0.113.42',
      },
    },
  });

  assert.equal(normalized.sourceIp, '203.0.113.42');
});

test('non-faucet requests do not need source IP attached to normalized payload', () => {
  const normalized = normalizeEvent({
    rawPath: '/v1/status',
    requestContext: {
      http: {
        method: 'GET',
        sourceIp: '203.0.113.42',
      },
    },
  });

  assert.equal(normalized.sourceIp, undefined);
});
