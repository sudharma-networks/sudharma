import test from 'node:test';
import assert from 'node:assert/strict';
import { proxyWithFailover, UpstreamUnavailableError } from './upstream.mjs';

const seeds = ['http://172.31.10.171:29100', 'http://172.31.32.195:29100'];

function response(status, body = '{}', headers = { 'content-type': 'application/json' }) {
  return new Response(body, { status, headers });
}

test('read uses primary only when primary succeeds', async () => {
  const calls = [];
  const fetchImpl = async (url) => {
    calls.push(url);
    return response(200, '{"status":"ready"}');
  };
  const result = await proxyWithFailover({ method: 'GET', path: '/ready', body: Buffer.alloc(0), headers: {} }, { seeds, fetchImpl, timeoutMs: 100 });
  assert.equal(result.statusCode, 200);
  assert.equal(calls.length, 1);
  assert.match(calls[0], /^http:\/\/172\.31\.10\.171:29100\/ready$/);
});

test('read fails over on transport failure and retryable 5xx', async () => {
  for (const first of ['transport', '5xx']) {
    const calls = [];
    const fetchImpl = async (url) => {
      calls.push(url);
      if (calls.length === 1) {
        if (first === 'transport') throw new TypeError('connect failed');
        return response(503, '{"error":"unavailable"}');
      }
      return response(200, '{"network":"sudharma"}');
    };
    const result = await proxyWithFailover({ method: 'GET', path: '/v1/status', body: Buffer.alloc(0), headers: {} }, { seeds, fetchImpl, timeoutMs: 100 });
    assert.equal(result.statusCode, 200, first);
    assert.equal(calls.length, 2, first);
    assert.match(calls[1], /^http:\/\/172\.31\.32\.195:29100\/v1\/status$/);
  }
});

test('authoritative 4xx does not fail over', async () => {
  let calls = 0;
  const fetchImpl = async () => {
    calls++;
    return response(404, '{"error":"not found"}');
  };
  const result = await proxyWithFailover({ method: 'GET', path: '/v1/transactions/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', body: Buffer.alloc(0), headers: {} }, { seeds, fetchImpl, timeoutMs: 100 });
  assert.equal(result.statusCode, 404);
  assert.equal(calls, 1);
});

test('transaction retry resends byte-for-byte identical signed body', async () => {
  const signed = Buffer.from('{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","nonce":7,"signature":"fixture-signature"}', 'utf8');
  const bodies = [];
  const fetchImpl = async (_url, init) => {
    bodies.push(Buffer.from(init.body));
    if (bodies.length === 1) throw new TypeError('socket closed before response');
    return response(202, '{"accepted":true,"transaction_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}');
  };
  const result = await proxyWithFailover({ method: 'POST', path: '/v1/transactions', body: signed, headers: { 'content-type': 'application/json' } }, { seeds, fetchImpl, timeoutMs: 100 });
  assert.equal(result.statusCode, 202);
  assert.equal(bodies.length, 2);
  assert.deepEqual(bodies[0], signed);
  assert.deepEqual(bodies[1], signed);
  assert.deepEqual(bodies[1], bodies[0]);
});

test('transaction authoritative 4xx is returned without replacement retry', async () => {
  const signed = Buffer.from('{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}');
  let calls = 0;
  const fetchImpl = async () => {
    calls++;
    return response(422, '{"error":"invalid transaction"}');
  };
  const result = await proxyWithFailover({ method: 'POST', path: '/v1/transactions', body: signed, headers: { 'content-type': 'application/json' } }, { seeds, fetchImpl, timeoutMs: 100 });
  assert.equal(result.statusCode, 422);
  assert.equal(calls, 1);
});

test('all seeds unavailable returns uncertain error and never claims success', async () => {
  let calls = 0;
  const fetchImpl = async () => {
    calls++;
    throw new TypeError('network unavailable');
  };
  await assert.rejects(
    proxyWithFailover({ method: 'POST', path: '/v1/transactions', body: Buffer.from('{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}'), headers: { 'content-type': 'application/json' } }, { seeds, fetchImpl, timeoutMs: 100 }),
    UpstreamUnavailableError,
  );
  assert.equal(calls, 2);
});

test('response filters unsafe upstream headers and forces no-store', async () => {
  const fetchImpl = async () => new Response('{"ok":true}', {
    status: 200,
    headers: {
      'content-type': 'application/json',
      'server': 'nginx/private',
      'set-cookie': 'secret=1',
      'cache-control': 'public, max-age=3600',
    },
  });
  const result = await proxyWithFailover({ method: 'GET', path: '/health', body: Buffer.alloc(0), headers: {} }, { seeds, fetchImpl, timeoutMs: 100 });
  assert.equal(result.headers['cache-control'], 'no-store');
  assert.equal(result.headers['content-type'], 'application/json');
  assert.equal(result.headers.server, undefined);
  assert.equal(result.headers['set-cookie'], undefined);
});
