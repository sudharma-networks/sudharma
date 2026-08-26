import test from 'node:test';
import assert from 'node:assert/strict';
import { createRuntimeFaucetHandler } from './faucet-runtime.mjs';

test('runtime fails closed when AWS faucet configuration is absent', () => {
  assert.throws(
    () => createRuntimeFaucetHandler({
      seeds: ['http://127.0.0.1:1', 'http://127.0.0.1:2'],
      env: {},
    }),
    /AWS configuration is incomplete/,
  );
});
