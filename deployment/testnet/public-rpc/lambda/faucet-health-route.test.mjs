import test from 'node:test';
import assert from 'node:assert/strict';
import { createHandler } from './index.mjs';
import { matchRoute } from './router.mjs';

test('faucet deep health is GET-only and routed locally', () => {
  assert.equal(matchRoute('GET', '/v1/faucet/health').kind, 'faucetHealth');
  assert.throws(() => matchRoute('POST', '/v1/faucet/health'), /method not allowed/i);
});

test('faucet deep health dispatches to protected faucet runtime instead of a seed', async () => {
  let upstreamCalls = 0;
  const handler = createHandler({
    fetchImpl: async () => {
      upstreamCalls += 1;
      throw new Error('health route must not proxy to seed');
    },
    faucetHandler: async (request) => {
      assert.equal(request.kind, 'faucetHealth');
      return { statusCode: 200, payload: { ready: true } };
    },
  });

  const response = await handler({
    rawPath: '/v1/faucet/health',
    requestContext: { http: { method: 'GET' } },
  }, { awsRequestId: 'test-health' });

  assert.equal(response.statusCode, 200);
  assert.deepEqual(JSON.parse(response.body), { ready: true });
  assert.equal(upstreamCalls, 0);
});
