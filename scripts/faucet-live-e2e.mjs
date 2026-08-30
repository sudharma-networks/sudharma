import { randomBytes } from 'node:crypto';
import { createSigner, COIN } from '../deployment/testnet/public-rpc/lambda/faucet.mjs';
import {
  DEVELOPMENT_TREASURY_ADDRESS,
  calculateFee,
  developmentFee,
  sudhToBaseUnits,
} from './faucet-fee-utils.mjs';

const DEFAULT_RPC_BASE_URL = 'https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com';
const DEFAULT_CONFIRM_TIMEOUT_MS = 8 * 60 * 1000;
const DEFAULT_POLL_INTERVAL_MS = 10_000;

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function fetchJson(url, init = {}) {
  const response = await fetch(url, init);
  const bodyText = await response.text();
  let body;
  try {
    body = bodyText ? JSON.parse(bodyText) : null;
  } catch {
    throw new Error(`invalid JSON from ${url}: ${bodyText.slice(0, 200)}`);
  }
  return { status: response.status, body };
}

export async function waitForFaucetEnabled(rpcBaseUrl, timeoutMs = 120_000) {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    const { status, body } = await fetchJson(`${rpcBaseUrl}/v1/faucet/info`);
    if (status === 200 && body?.enabled === true) return body;
    await sleep(3000);
  }
  throw new Error('faucet did not become enabled');
}

export async function getAccountBalance(rpcBaseUrl, address) {
  const { status, body } = await fetchJson(`${rpcBaseUrl}/v1/accounts/${address}`);
  if (status !== 200) {
    throw new Error(`account lookup failed for ${address}: HTTP ${status}`);
  }
  return body.balance;
}

export async function waitForTransactionConfirmed(rpcBaseUrl, transactionId, timeoutMs, pollIntervalMs) {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    const { status, body } = await fetchJson(`${rpcBaseUrl}/v1/transactions/${transactionId}`);
    if (status === 200 && body?.status === 'confirmed' && Number(body?.confirmations || 0) >= 1) {
      return body;
    }
    await sleep(pollIntervalMs);
  }
  throw new Error(`transaction ${transactionId} was not confirmed within ${timeoutMs}ms`);
}

export async function assertTreasuryReceivedDevelopmentFee({
  rpcBaseUrl,
  balanceBefore,
  amount,
  label,
}) {
  const balanceAfter = await getAccountBalance(rpcBaseUrl, DEVELOPMENT_TREASURY_ADDRESS);
  const expectedIncrease = developmentFee(amount);
  const actualIncrease = balanceAfter - balanceBefore;
  if (actualIncrease < expectedIncrease) {
    throw new Error(
      `${label}: development treasury increase ${actualIncrease} is below expected ${expectedIncrease}`,
    );
  }
  return {
    treasury_address: DEVELOPMENT_TREASURY_ADDRESS,
    expected_development_fee: expectedIncrease,
    actual_increase: actualIncrease,
    balance_before: balanceBefore,
    balance_after: balanceAfter,
  };
}

export async function runFaucetLiveE2E({
  rpcBaseUrl = process.env.RPC_BASE_URL || DEFAULT_RPC_BASE_URL,
  confirmTimeoutMs = Number.parseInt(process.env.FAUCET_E2E_CONFIRM_TIMEOUT_MS || '', 10) || DEFAULT_CONFIRM_TIMEOUT_MS,
  pollIntervalMs = Number.parseInt(process.env.FAUCET_E2E_POLL_INTERVAL_MS || '', 10) || DEFAULT_POLL_INTERVAL_MS,
  walletScalarHex = randomBytes(32).toString('hex'),
  now = Date.now,
} = {}) {
  const wallet = createSigner(walletScalarHex);
  const info = await waitForFaucetEnabled(rpcBaseUrl);
  const challengeAddress = info.challenge_address;
  if (typeof challengeAddress !== 'string' || challengeAddress.length !== 40) {
    throw new Error('faucet info missing challenge_address');
  }

  const grantAmount = sudhToBaseUnits(info.initial_grant_sudh || 100);
  const challengeSendAmount = sudhToBaseUnits(info.challenge_send_sudh || 25);
  const challengeRewardAmount = sudhToBaseUnits(info.challenge_reward_sudh || 50);

  let treasuryBalance = await getAccountBalance(rpcBaseUrl, DEVELOPMENT_TREASURY_ADDRESS);

  const grantResponse = await fetchJson(`${rpcBaseUrl}/v1/faucet/request`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ address: wallet.address }),
  });
  if (grantResponse.status !== 202) {
    throw new Error(`initial grant failed: HTTP ${grantResponse.status} ${JSON.stringify(grantResponse.body)}`);
  }
  const grantTxId = grantResponse.body?.transaction_id;
  if (typeof grantTxId !== 'string') {
    throw new Error('initial grant response missing transaction_id');
  }

  const grantStatus = await waitForTransactionConfirmed(
    rpcBaseUrl,
    grantTxId,
    confirmTimeoutMs,
    pollIntervalMs,
  );
  const grantTx = grantStatus.transaction;
  if (!grantTx || grantTx.To !== wallet.address || grantTx.Amount !== grantAmount) {
    throw new Error('confirmed initial grant transaction payload mismatch');
  }
  if (grantTx.Fee !== calculateFee(grantAmount)) {
    throw new Error('initial grant transaction fee mismatch');
  }
  const grantTreasury = await assertTreasuryReceivedDevelopmentFee({
    rpcBaseUrl,
    balanceBefore: treasuryBalance,
    amount: grantAmount,
    label: 'initial grant',
  });
  treasuryBalance = grantTreasury.balance_after;

  const walletBalance = await getAccountBalance(rpcBaseUrl, wallet.address);
  if (walletBalance < grantAmount) {
    throw new Error(`wallet balance ${walletBalance} is below grant amount ${grantAmount}`);
  }

  const challengePayment = wallet.signTransaction(challengeAddress, challengeSendAmount, 0);
  if (challengePayment.Fee !== calculateFee(challengeSendAmount)) {
    throw new Error('challenge payment fee mismatch');
  }
  const submitChallenge = await fetchJson(`${rpcBaseUrl}/v1/transactions`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify(challengePayment),
  });
  if (submitChallenge.status !== 202 && submitChallenge.status !== 200) {
    throw new Error(`challenge payment submit failed: HTTP ${submitChallenge.status}`);
  }

  const challengeStatus = await waitForTransactionConfirmed(
    rpcBaseUrl,
    challengePayment.ID,
    confirmTimeoutMs,
    pollIntervalMs,
  );
  const challengeTx = challengeStatus.transaction;
  if (!challengeTx || challengeTx.From !== wallet.address || challengeTx.To !== challengeAddress) {
    throw new Error('confirmed challenge payment payload mismatch');
  }
  const challengeTreasury = await assertTreasuryReceivedDevelopmentFee({
    rpcBaseUrl,
    balanceBefore: treasuryBalance,
    amount: challengeSendAmount,
    label: 'challenge payment',
  });
  treasuryBalance = challengeTreasury.balance_after;

  const claimResponse = await fetchJson(`${rpcBaseUrl}/v1/faucet/challenge`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      address: wallet.address,
      transaction_id: challengePayment.ID,
    }),
  });
  if (claimResponse.status !== 202) {
    throw new Error(`challenge claim failed: HTTP ${claimResponse.status} ${JSON.stringify(claimResponse.body)}`);
  }
  const rewardTxId = claimResponse.body?.reward_transaction_id;
  if (typeof rewardTxId !== 'string') {
    throw new Error('challenge claim response missing reward_transaction_id');
  }

  const rewardStatus = await waitForTransactionConfirmed(
    rpcBaseUrl,
    rewardTxId,
    confirmTimeoutMs,
    pollIntervalMs,
  );
  const rewardTx = rewardStatus.transaction;
  if (!rewardTx || rewardTx.To !== wallet.address || rewardTx.Amount !== challengeRewardAmount) {
    throw new Error('confirmed challenge reward payload mismatch');
  }
  const rewardTreasury = await assertTreasuryReceivedDevelopmentFee({
    rpcBaseUrl,
    balanceBefore: treasuryBalance,
    amount: challengeRewardAmount,
    label: 'challenge reward',
  });

  return {
    ok: true,
    wallet_address: wallet.address,
    challenge_address: challengeAddress,
    initial_grant: {
      transaction_id: grantTxId,
      amount_sudh: grantAmount / COIN,
      development_fee: grantTreasury.expected_development_fee,
      treasury_increase: grantTreasury.actual_increase,
    },
    challenge_payment: {
      transaction_id: challengePayment.ID,
      amount_sudh: challengeSendAmount / COIN,
      development_fee: challengeTreasury.expected_development_fee,
      treasury_increase: challengeTreasury.actual_increase,
    },
    challenge_reward: {
      transaction_id: rewardTxId,
      amount_sudh: challengeRewardAmount / COIN,
      round: claimResponse.body?.round,
      development_fee: rewardTreasury.expected_development_fee,
      treasury_increase: rewardTreasury.actual_increase,
    },
    development_treasury_address: DEVELOPMENT_TREASURY_ADDRESS,
    completed_at_ms: now(),
  };
}

if (process.argv[1]?.endsWith('faucet-live-e2e.mjs')) {
  runFaucetLiveE2E()
    .then((result) => {
      process.stdout.write(`${JSON.stringify(result)}\n`);
    })
    .catch((error) => {
      process.stderr.write(`${error?.stack || error}\n`);
      process.exit(1);
    });
}
