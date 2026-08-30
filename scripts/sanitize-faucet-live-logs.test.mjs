import test from 'node:test';
import assert from 'node:assert/strict';
import { sanitizeLiveLogText } from './sanitize-faucet-live-logs.mjs';

test('live log sanitizer keeps only allowlisted faucet diagnostic fields', () => {
  const dumped = [
    '2026-08-30T15:00:00Z\tINFO\t{"event":"faucet_dependency","operation":"seed.submit_transaction","outcome":"error","error_name":"FaucetError","http_status":422,"error_category":"invalid_signature","latency_ms":12,"error":"transaction rejected by mempool: invalid transaction signature","body":"signed-transaction-hex","signature":"aabbcc"}',
    '{"event":"wallet_faucet_error","route":"faucetInitial","status_code":503,"request_id":"abc-123","message":"private-sensitive-marker"}',
    '{"event":"wallet_proxy_request","path":"/v1/faucet/initial","body":"should-not-pass"}',
    'not json at all',
  ].join('\n');

  assert.deepEqual(sanitizeLiveLogText(dumped), [
    {
      event: 'faucet_dependency',
      operation: 'seed.submit_transaction',
      outcome: 'error',
      error_name: 'FaucetError',
      http_status: 422,
      error_category: 'invalid_signature',
      latency_ms: 12,
    },
    {
      event: 'wallet_faucet_error',
      route: 'faucetInitial',
      status_code: 503,
      request_id: 'abc-123',
    },
  ]);
  assert.equal(JSON.stringify(sanitizeLiveLogText(dumped)).includes('private-sensitive-marker'), false);
  assert.equal(JSON.stringify(sanitizeLiveLogText(dumped)).includes('signed-transaction-hex'), false);
});

test('live log sanitizer parses Node inspect CloudWatch messages without extra fields', () => {
  const dumped = [
    "2026-08-30T12:56:47.513Z\t634ac672-71c7-4afb-ab60-272ed93870ca\tINFO\t{",
    "  event: 'faucet_dependency',",
    "  operation: 'seed.submit_transaction',",
    "  outcome: 'success',",
    "  latency_ms: 161,",
    "  error: 'transaction rejected by mempool: invalid transaction signature'",
    '}',
    "2026-08-30T12:56:47.792Z\t634ac672-71c7-4afb-ab60-272ed93870ca\tERROR\t{",
    "  event: 'wallet_faucet_error',",
    "  route: 'faucetInitial',",
    '  status_code: 503,',
    "  message: 'private-sensitive-marker'",
    '}',
  ].join('\n');

  assert.deepEqual(sanitizeLiveLogText(dumped), [
    {
      request_id: '634ac672-71c7-4afb-ab60-272ed93870ca',
      cw_timestamp: '2026-08-30T12:56:47.513Z',
      event: 'faucet_dependency',
      operation: 'seed.submit_transaction',
      outcome: 'success',
      latency_ms: 161,
    },
    {
      request_id: '634ac672-71c7-4afb-ab60-272ed93870ca',
      cw_timestamp: '2026-08-30T12:56:47.792Z',
      event: 'wallet_faucet_error',
      route: 'faucetInitial',
      status_code: 503,
    },
  ]);
  assert.equal(JSON.stringify(sanitizeLiveLogText(dumped)).includes('private-sensitive-marker'), false);
  assert.equal(JSON.stringify(sanitizeLiveLogText(dumped)).includes('transaction rejected by mempool'), false);
});

test('live log sanitizer keeps CloudWatch request id without raw seed bodies', () => {
  const dumped = [
    "2026-08-30T12:56:47.513Z\t634ac672-71c7-4afb-ab60-272ed93870ca\tINFO\t{",
    "  event: 'faucet_dependency',",
    "  operation: 'seed.submit_transaction',",
    "  outcome: 'success',",
    "  latency_ms: 161",
    '}',
  ].join('\n');

  assert.deepEqual(sanitizeLiveLogText(dumped), [
    {
      request_id: '634ac672-71c7-4afb-ab60-272ed93870ca',
      cw_timestamp: '2026-08-30T12:56:47.513Z',
      event: 'faucet_dependency',
      operation: 'seed.submit_transaction',
      outcome: 'success',
      latency_ms: 161,
    },
  ]);
});
