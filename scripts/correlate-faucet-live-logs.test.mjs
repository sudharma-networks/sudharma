import test from 'node:test';
import assert from 'node:assert/strict';
import { correlateFaucetLiveLogs } from './correlate-faucet-live-logs.mjs';

test('correlator keeps only allowlisted records for one request id', () => {
  const dumped = [
    "2026-08-30T12:56:47.112Z\t634ac672-71c7-4afb-ab60-272ed93870ca\tINFO\t{",
    "  event: 'faucet_dependency',",
    "  operation: 'dynamodb.reserve_initial',",
    "  outcome: 'success',",
    "  latency_ms: 519",
    '}',
    "2026-08-30T12:56:47.513Z\t634ac672-71c7-4afb-ab60-272ed93870ca\tINFO\t{",
    "  event: 'faucet_dependency',",
    "  operation: 'seed.submit_transaction',",
    "  outcome: 'success',",
    "  latency_ms: 161",
    '}',
    "2026-08-30T12:56:47.792Z\t634ac672-71c7-4afb-ab60-272ed93870ca\tERROR\t{",
    "  event: 'wallet_faucet_error',",
    "  route: 'faucetInitial',",
    '  status_code: 503,',
    '}',
    "2026-08-30T12:57:40.192Z\tabf734be-a125-4b79-9355-05ce61a6b492\tINFO\t{",
    "  event: 'faucet_dependency',",
    "  operation: 'dynamodb.reserve_initial',",
    "  outcome: 'error',",
    "  error_name: 'ConditionalCheckFailedException',",
    "  latency_ms: 63",
    '}',
  ].join('\n');

  assert.deepEqual(
    correlateFaucetLiveLogs(dumped, '634ac672-71c7-4afb-ab60-272ed93870ca'),
    [
      {
        request_id: '634ac672-71c7-4afb-ab60-272ed93870ca',
        cw_timestamp: '2026-08-30T12:56:47.112Z',
        event: 'faucet_dependency',
        operation: 'dynamodb.reserve_initial',
        outcome: 'success',
        latency_ms: 519,
      },
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
    ],
  );
});
