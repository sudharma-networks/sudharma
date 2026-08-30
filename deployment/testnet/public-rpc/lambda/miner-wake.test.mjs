import test from 'node:test';
import assert from 'node:assert/strict';

import { wakeDemandMiner, wakeDemandMinerInBackground } from './miner-wake.mjs';

test('wakeDemandMiner posts to the miner wake route on the first healthy seed', async () => {
  const calls = [];
  const fetchImpl = async (url, init) => ({
    status: 202,
    headers: { get: () => 'application/json' },
    arrayBuffer: async () => Buffer.from('{"awoken":true}'),
  });

  await wakeDemandMiner({
    seeds: ['http://172.31.10.171:29100', 'http://172.31.32.195:29100'],
    fetchImpl: async (url, init) => {
      calls.push({ url, method: init.method });
      return fetchImpl(url, init);
    },
    timeoutMs: 1000,
  });

  assert.equal(calls.length, 1);
  assert.match(calls[0].url, /\/v1\/miner\/wake$/);
  assert.equal(calls[0].method, 'POST');
});

test('wakeDemandMinerInBackground does not throw when wake fails', async () => {
  const warnings = [];
  wakeDemandMinerInBackground({
    seeds: ['http://172.31.10.171:29100', 'http://172.31.32.195:29100'],
    fetchImpl: async () => {
      throw new Error('seed unreachable');
    },
    timeoutMs: 100,
  }, {
    warn: (record) => warnings.push(record),
  });
  await new Promise((resolve) => setTimeout(resolve, 20));
  assert.equal(warnings.length, 1);
  assert.equal(warnings[0].event, 'demand_miner_wake_failed');
});
