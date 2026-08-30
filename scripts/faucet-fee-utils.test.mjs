import test from 'node:test';
import assert from 'node:assert/strict';
import {
  COIN,
  assertFeeSplit,
  calculateFee,
  developmentFee,
  expectedTreasuryIncreaseForAmounts,
  miningFee,
  sudhToBaseUnits,
} from './faucet-fee-utils.mjs';

test('100 SUDH faucet grant fee split matches consensus rules', () => {
  const amount = sudhToBaseUnits(100);
  const split = assertFeeSplit(amount);
  assert.equal(split.total, 10_000_000);
  assert.equal(split.development, 1_000_000);
  assert.equal(split.mining, 9_000_000);
});

test('25 SUDH challenge payment credits development treasury with 0.01%', () => {
  const amount = sudhToBaseUnits(25);
  assert.equal(developmentFee(amount), 250_000);
  assert.equal(miningFee(amount), 2_250_000);
  assert.equal(calculateFee(amount), 2_500_000);
});

test('50 SUDH challenge reward credits development treasury with 0.01%', () => {
  const amount = sudhToBaseUnits(50);
  assert.equal(developmentFee(amount), 500_000);
});

test('full faucet round treasury increase sums development fees only', () => {
  const increase = expectedTreasuryIncreaseForAmounts([
    sudhToBaseUnits(100),
    sudhToBaseUnits(25),
    sudhToBaseUnits(50),
  ]);
  assert.equal(increase, 1_000_000 + 250_000 + 500_000);
  assert.equal(increase / COIN, 0.0175);
});
