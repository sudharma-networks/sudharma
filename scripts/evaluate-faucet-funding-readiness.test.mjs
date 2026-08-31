import test from 'node:test';
import assert from 'node:assert/strict';
import {
  evaluateFaucetFundingReadiness,
  requiredInitialGrantBalance,
} from './evaluate-faucet-funding-readiness.mjs';

test('initial grant requires 100.01 SUDH including fee', () => {
  assert.equal(requiredInitialGrantBalance(), 10_010_000_000);
});

test('funding readiness flags low signer balance', () => {
  const result = evaluateFaucetFundingReadiness({
    faucetInfo: { enabled: true },
    faucetDiagnostics: { ready: false },
    signerAccount: { balance: 7_496_250_000 },
  });
  assert.equal(result.should_refill, true);
  assert.equal(result.reason, 'faucet_needs_funding');
  assert.ok(result.shortfall > 0);
});

test('funding readiness passes when signer can fund one grant', () => {
  const result = evaluateFaucetFundingReadiness({
    faucetInfo: { enabled: true },
    faucetDiagnostics: { ready: true },
    signerAccount: { balance: 10_010_000_000 },
  });
  assert.equal(result.ready, true);
  assert.equal(result.should_refill, false);
});
