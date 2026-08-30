import test from 'node:test';
import assert from 'node:assert/strict';
import { evaluateFaucetRecoveryReadiness } from './evaluate-faucet-recovery-readiness.mjs';

test('recovery readiness skips while mempool nonce conflict remains', () => {
  const result = evaluateFaucetRecoveryReadiness({
    failed_payout: { chain_status: 'not_found' },
    likely_blocker: 'mempool_nonce_conflict',
    network_mempool: 2,
    last_error_category: 'invalid_nonce',
  });
  assert.equal(result.should_attempt_recovery, false);
  assert.equal(result.reason, 'mempool_nonce_conflict');
});

test('recovery readiness allows retry when blocker cleared', () => {
  const result = evaluateFaucetRecoveryReadiness({
    failed_payout: { chain_status: 'not_found' },
    likely_blocker: 'submit_rejected_not_on_chain',
    network_mempool: 0,
  });
  assert.equal(result.should_attempt_recovery, true);
  assert.equal(result.reason, 'ready_to_retry');
});
