export const COIN = 100_000_000;
export const DEVELOPMENT_TREASURY_ADDRESS = '16d7dc9ec0495109007860a584c7cf9055da9abf';
export const DEVELOPMENT_FEE_BASIS_POINTS = 1;
export const MINING_FEE_BASIS_POINTS = 9;
export const TOTAL_FEE_BASIS_POINTS = 10;

export function sudhToBaseUnits(sudh) {
  if (!Number.isSafeInteger(sudh) || sudh <= 0) {
    throw new Error('sudh amount must be a positive integer');
  }
  return sudh * COIN;
}

export function calculateFee(amount) {
  return Math.floor((amount * TOTAL_FEE_BASIS_POINTS) / 10_000);
}

export function developmentFee(amount) {
  return Math.floor((amount * DEVELOPMENT_FEE_BASIS_POINTS) / 10_000);
}

export function miningFee(amount) {
  return Math.floor((amount * MINING_FEE_BASIS_POINTS) / 10_000);
}

export function expectedTreasuryIncreaseForAmounts(amounts) {
  return amounts.reduce((sum, amount) => sum + developmentFee(amount), 0);
}

export function assertFeeSplit(amount) {
  const total = calculateFee(amount);
  const development = developmentFee(amount);
  const mining = miningFee(amount);
  if (development + mining !== total) {
    throw new Error(`fee split mismatch for amount ${amount}`);
  }
  return { total, development, mining };
}
