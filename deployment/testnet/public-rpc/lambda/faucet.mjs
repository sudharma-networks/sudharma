import {
  createECDH,
  createHash,
  createPrivateKey,
  sign as cryptoSign,
} from 'node:crypto';

export const COIN = 100_000_000;
export const INITIAL_GRANT_SUDH = 100;
export const CHALLENGE_SEND_SUDH = 25;
export const CHALLENGE_REWARD_SUDH = 50;
export const MAX_ROUNDS = 5;
export const COOLDOWN_MS = 24 * 60 * 60 * 1000;

const LOWER_HEX_40 = /^[0-9a-f]{40}$/;
const LOWER_HEX_64 = /^[0-9a-f]{64}$/;
const PRIVATE_SCALAR_HEX = /^[0-9a-f]{64}$/;

export class FaucetError extends Error {
  constructor(statusCode, message) {
    super(message);
    this.name = 'FaucetError';
    this.statusCode = statusCode;
  }
}

function sha256(data) {
  return createHash('sha256').update(data).digest();
}

function base64url(bytes) {
  return Buffer.from(bytes).toString('base64url');
}

function validateAddress(address) {
  if (typeof address !== 'string' || !LOWER_HEX_40.test(address)) {
    throw new FaucetError(400, 'invalid Sudharma address');
  }
}

function validateTransactionId(transactionId) {
  if (typeof transactionId !== 'string' || !LOWER_HEX_64.test(transactionId)) {
    throw new FaucetError(400, 'invalid transaction ID');
  }
}

export function createSigner(privateScalarHex) {
  if (typeof privateScalarHex !== 'string' || !PRIVATE_SCALAR_HEX.test(privateScalarHex)) {
    throw new Error('faucet private scalar must be exactly 32 lowercase-hex bytes');
  }

  const privateBytes = Buffer.from(privateScalarHex, 'hex');
  if (privateBytes.every((value) => value === 0)) {
    throw new Error('faucet private scalar cannot be zero');
  }

  const ecdh = createECDH('prime256v1');
  try {
    ecdh.setPrivateKey(privateBytes);
  } catch {
    throw new Error('faucet private scalar is outside the P-256 key range');
  }

  const publicKey = ecdh.getPublicKey(undefined, 'uncompressed');
  const address = sha256(publicKey).subarray(0, 20).toString('hex');
  const x = publicKey.subarray(1, 33);
  const y = publicKey.subarray(33, 65);
  const privateKey = createPrivateKey({
    key: {
      kty: 'EC',
      crv: 'P-256',
      d: base64url(privateBytes),
      x: base64url(x),
      y: base64url(y),
    },
    format: 'jwk',
  });

  return {
    address,
    signTransaction(to, amount, nonce) {
      validateAddress(to);
      if (!Number.isSafeInteger(amount) || amount <= 0) throw new Error('invalid transaction amount');
      if (!Number.isSafeInteger(nonce) || nonce < 0) throw new Error('invalid transaction nonce');

      const fee = Math.floor((amount * 10) / 10_000);
      const canonical = `${address}|${to}|${amount}|${fee}|${nonce}`;
      const id = sha256(Buffer.from(canonical, 'utf8')).toString('hex');
      const signature = cryptoSign('sha256', Buffer.from(id, 'utf8'), {
        key: privateKey,
        dsaEncoding: 'ieee-p1363',
      });

      return {
        ID: id,
        From: address,
        To: to,
        Amount: amount,
        Fee: fee,
        Nonce: nonce,
        PublicKey: publicKey.toString('base64'),
        Signature: signature.toString('base64'),
      };
    },
  };
}

function payoutTxId(result, fallback) {
  return result?.transaction_id || result?.transactionId || fallback;
}

function initialGrantResult(address, transactionId, status) {
  return {
    address,
    amount_sudh: INITIAL_GRANT_SUDH,
    transaction_id: transactionId,
    status,
  };
}

async function reconcileInitialGrant({ store, rpc, address, state, now }) {
  const transactionId = state?.initial_txid;
  if (
    (state?.initial_status !== 'submitted' && state?.initial_status !== 'paid') ||
    typeof transactionId !== 'string' ||
    !LOWER_HEX_64.test(transactionId)
  ) {
    return null;
  }

  const remote = await rpc.transaction(transactionId);
  const confirmed = remote?.status === 'confirmed' && Number(remote?.confirmations || 0) >= 1;
  if (confirmed) {
    if (state.initial_status === 'submitted') {
      await store.completeInitial(address, transactionId, now());
    }
    return initialGrantResult(address, transactionId, 'confirmed');
  }

  if (state.initial_status === 'paid' && typeof store.markInitialSubmitted === 'function') {
    await store.markInitialSubmitted(address, transactionId, now());
  }
  return initialGrantResult(address, transactionId, 'submitted');
}

async function submitPayout({ store, rpc, signer, to, amount }) {
  const locked = await store.acquirePayoutLock();
  if (!locked) throw new FaucetError(503, 'faucet is busy; retry shortly');

  try {
    const account = await rpc.account(signer.address);
    const nonce = account?.next_nonce ?? account?.nextNonce;
    const balance = account?.balance;
    if (!Number.isSafeInteger(nonce) || nonce < 0) {
      throw new FaucetError(503, 'faucet account nonce is unavailable');
    }

    const tx = signer.signTransaction(to, amount, nonce);
    if (!Number.isSafeInteger(balance) || balance < tx.Amount + tx.Fee) {
      throw new FaucetError(503, 'testnet faucet needs funding');
    }

    const result = await rpc.submit(tx);
    if (!result?.accepted) throw new FaucetError(503, 'faucet payout was not accepted');
    return payoutTxId(result, tx.ID);
  } finally {
    await store.releasePayoutLock();
  }
}

export function createFaucetService({ store, rpc, signer, now = Date.now }) {
  if (!store || !rpc || !signer) throw new Error('faucet service dependencies are required');

  return {
    async requestInitial(address) {
      validateAddress(address);
      const reserved = await store.reserveInitial(address, now());
      if (!reserved) {
        const state = await store.getAddress(address);
        const reconciled = await reconcileInitialGrant({ store, rpc, address, state, now });
        if (reconciled) return reconciled;
        throw new FaucetError(409, 'initial 100 SUDH grant was already requested for this address');
      }

      try {
        const transactionId = await submitPayout({
          store,
          rpc,
          signer,
          to: address,
          amount: INITIAL_GRANT_SUDH * COIN,
        });
        await store.markInitialSubmitted(address, transactionId, now());
        return initialGrantResult(address, transactionId, 'submitted');
      } catch (error) {
        if (!error?.uncertain && typeof store.failInitial === 'function') {
          await store.failInitial(address, String(error?.message || error));
        }
        throw error;
      }
    },

    async claimChallenge(address, transactionId) {
      validateAddress(address);
      validateTransactionId(transactionId);

      let state = await store.getAddress(address);
      if (state?.initial_status === 'submitted') {
        const reconciled = await reconcileInitialGrant({ store, rpc, address, state, now });
        if (reconciled?.status === 'confirmed') {
          state = { ...state, initial_status: 'paid' };
        }
      }
      if (!state || state.initial_status !== 'paid') {
        throw new FaucetError(409, 'request the initial 100 SUDH test grant before joining the challenge');
      }

      const rounds = Number(state.rounds || 0);
      if (rounds >= MAX_ROUNDS) {
        throw new FaucetError(409, 'maximum 5 challenge rounds already completed');
      }

      const currentTime = now();
      if (state.last_round_at != null && currentTime - Number(state.last_round_at) < COOLDOWN_MS) {
        throw new FaucetError(409, '24-hour cooldown has not finished yet');
      }

      const remote = await rpc.transaction(transactionId);
      if (remote?.status !== 'confirmed' || Number(remote?.confirmations || 0) < 1) {
        throw new FaucetError(409, 'challenge transaction is not confirmed yet');
      }

      const tx = remote.transaction;
      if (!tx || tx.From !== address || tx.To !== signer.address || tx.Amount !== CHALLENGE_SEND_SUDH * COIN) {
        throw new FaucetError(422, 'transaction must send exactly 25 SUDH from this wallet to the official challenge address');
      }

      const round = rounds + 1;
      const reserved = await store.reserveChallenge(address, transactionId, round, currentTime);
      if (!reserved) throw new FaucetError(409, 'this challenge transaction was already claimed');

      try {
        const rewardTransactionId = await submitPayout({
          store,
          rpc,
          signer,
          to: address,
          amount: CHALLENGE_REWARD_SUDH * COIN,
        });
        const completedAt = now();
        await store.completeChallenge(address, transactionId, rewardTransactionId, completedAt);
        return {
          address,
          round,
          reward_sudh: CHALLENGE_REWARD_SUDH,
          reward_transaction_id: rewardTransactionId,
          next_eligible_at: round < MAX_ROUNDS ? completedAt + COOLDOWN_MS : null,
          status: 'submitted',
        };
      } catch (error) {
        if (!error?.uncertain && typeof store.failChallenge === 'function') {
          await store.failChallenge(address, transactionId, String(error?.message || error));
        }
        throw error;
      }
    },
  };
}
