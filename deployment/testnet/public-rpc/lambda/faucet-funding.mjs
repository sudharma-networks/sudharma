const COIN = 100_000_000;

export const INITIAL_GRANT_REQUIRED_BALANCE = (() => {
  const amount = 100 * COIN;
  const fee = Math.floor((amount * 10) / 10_000);
  return amount + fee;
})();

export class FaucetFundingError extends Error {
  constructor(statusCode, message) {
    super(message);
    this.name = 'FaucetFundingError';
    this.statusCode = statusCode;
  }
}

export function requiredPayoutBalance(amountBaseUnits) {
  const fee = Math.floor((amountBaseUnits * 10) / 10_000);
  return amountBaseUnits + fee;
}

export async function waitForFaucetFunding({
  rpc,
  signer,
  requiredBalance = INITIAL_GRANT_REQUIRED_BALANCE,
  timeoutMs = 180_000,
  pollMs = 5_000,
  sleep = defaultSleep,
  now = Date.now,
}) {
  const started = now();
  while (now() - started < timeoutMs) {
    const account = await rpc.account(signer.address);
    const balance = account?.balance;
    if (Number.isSafeInteger(balance) && balance >= requiredBalance) {
      return { funded: true, balance };
    }
    await sleep(pollMs);
  }
  throw new FaucetFundingError(503, 'testnet faucet is funding; retry shortly');
}

function defaultSleep(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}
