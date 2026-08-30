import { randomUUID } from 'node:crypto';
import { DynamoDBClient } from '@aws-sdk/client-dynamodb';
import {
  DeleteCommand,
  DynamoDBDocumentClient,
  GetCommand,
  PutCommand,
  TransactWriteCommand,
  UpdateCommand,
} from '@aws-sdk/lib-dynamodb';
import { GetSecretValueCommand, SecretsManagerClient } from '@aws-sdk/client-secrets-manager';
import {
  CHALLENGE_REWARD_SUDH,
  CHALLENGE_SEND_SUDH,
  COIN,
  COOLDOWN_MS,
  FaucetError,
  INITIAL_GRANT_SUDH,
  MAX_ROUNDS,
  createFaucetService,
  createSigner,
} from './faucet.mjs';
import { proxyWithFailover, fetchOnce } from './upstream.mjs';

function conditionalFailure(error) {
  return error?.name === 'ConditionalCheckFailedException' || error?.name === 'TransactionCanceledException';
}

function writeDiagnosticLog(logger, record) {
  const line = JSON.stringify(record);
  if (typeof logger?.info === 'function') logger.info(line);
}

export function createOperationTimer({ logger = console, now = Date.now } = {}) {
  return async function timeOperation(operation, action) {
    const started = now();
    try {
      const result = await action();
      writeDiagnosticLog(logger, { event: 'faucet_dependency', operation, outcome: 'success', latency_ms: now() - started });
      return result;
    } catch (error) {
      const record = {
        event: 'faucet_dependency',
        operation,
        outcome: 'error',
        error_name: String(error?.name || 'Error'),
        latency_ms: now() - started,
      };
      if (Number.isInteger(error?.upstreamStatus)) record.http_status = error.upstreamStatus;
      if (typeof error?.errorCategory === 'string') record.error_category = error.errorCategory;
      if (Number.isInteger(error?.expectedNonce)) record.expected_nonce = error.expectedNonce;
      if (Number.isInteger(error?.submittedNonce)) record.submitted_nonce = error.submittedNonce;
      writeDiagnosticLog(logger, record);
      throw error;
    }
  };
}

export function createStore(tableName, timed, clientOverride = null) {
  const client = clientOverride || DynamoDBDocumentClient.from(new DynamoDBClient({}), {
    marshallOptions: { removeUndefinedValues: true },
  });
  const lockOwner = randomUUID();
  const send = (operation, command) => timed(operation, () => client.send(command));

  return {
    async checkReadWrite() {
      const owner = randomUUID();
      const key = `HEALTH#${owner}`;
      await send('dynamodb.health_write', new PutCommand({
        TableName: tableName,
        Item: { pk: key, kind: 'health', owner, expires_at: Date.now() + 60_000 },
        ConditionExpression: 'attribute_not_exists(pk)',
      }));
      try {
        const result = await send('dynamodb.health_read', new GetCommand({
          TableName: tableName,
          Key: { pk: key },
          ConsistentRead: true,
        }));
        if (result.Item?.owner !== owner) {
          throw new FaucetError(503, 'faucet state store health check failed');
        }
      } finally {
        await send('dynamodb.health_delete', new DeleteCommand({
          TableName: tableName,
          Key: { pk: key },
          ConditionExpression: '#owner = :owner',
          ExpressionAttributeNames: { '#owner': 'owner' },
          ExpressionAttributeValues: { ':owner': owner },
        }));
      }
    },

    async getAddress(address) {
      const result = await send('dynamodb.get_address', new GetCommand({ TableName: tableName, Key: { pk: `ADDR#${address}` } }));
      return result.Item || null;
    },

    async voidAddress(address) {
      try {
        await send('dynamodb.void_address', new DeleteCommand({
          TableName: tableName,
          Key: { pk: `ADDR#${address}` },
        }));
      } catch (error) {
        if (error?.name !== 'ConditionalCheckFailedException') throw error;
      }
    },

    async getChallenge(transactionId) {
      const result = await send('dynamodb.get_challenge', new GetCommand({
        TableName: tableName,
        Key: { pk: `TX#${transactionId}` },
        ConsistentRead: true,
      }));
      return result.Item || null;
    },

    async reserveInitial(address, at) {
      try {
        await send('dynamodb.reserve_initial', new UpdateCommand({
          TableName: tableName,
          Key: { pk: `ADDR#${address}` },
          UpdateExpression: 'SET #kind = if_not_exists(#kind, :kind), address = :address, initial_status = :reserved, initial_reserved_at = :at, rounds = if_not_exists(rounds, :zero)',
          ConditionExpression: 'attribute_not_exists(pk) OR initial_status = :failed',
          ExpressionAttributeNames: { '#kind': 'kind' },
          ExpressionAttributeValues: {
            ':kind': 'address', ':address': address, ':reserved': 'reserved', ':failed': 'failed', ':at': at, ':zero': 0,
          },
        }));
        return true;
      } catch (error) {
        if (conditionalFailure(error)) return false;
        throw error;
      }
    },

    async prepareInitial(address, payout, at) {
      await send('dynamodb.prepare_initial', new UpdateCommand({
        TableName: tableName,
        Key: { pk: `ADDR#${address}` },
        UpdateExpression: 'SET initial_status = :prepared, initial_txid = :txid, initial_payout = :payout, initial_prepared_at = :at',
        ConditionExpression: 'initial_status = :reserved OR (initial_status = :prepared AND initial_txid = :txid)',
        ExpressionAttributeValues: {
          ':prepared': 'prepared', ':reserved': 'reserved', ':txid': payout.ID, ':payout': payout, ':at': at,
        },
      }));
    },

    async markInitialSubmitted(address, transactionId, at) {
      await send('dynamodb.mark_initial_submitted', new UpdateCommand({
        TableName: tableName,
        Key: { pk: `ADDR#${address}` },
        UpdateExpression: 'SET initial_status = :submitted, initial_txid = :txid, initial_submitted_at = :at',
        ConditionExpression: 'initial_status = :reserved OR (initial_status = :prepared AND initial_txid = :txid) OR (initial_status = :paid AND initial_txid = :txid)',
        ExpressionAttributeValues: {
          ':submitted': 'submitted', ':reserved': 'reserved', ':prepared': 'prepared', ':paid': 'paid', ':txid': transactionId, ':at': at,
        },
      }));
    },

    async completeInitial(address, transactionId, at) {
      await send('dynamodb.complete_initial', new UpdateCommand({
        TableName: tableName,
        Key: { pk: `ADDR#${address}` },
        UpdateExpression: 'SET initial_status = :paid, initial_txid = :txid, initial_paid_at = :at REMOVE initial_payout',
        ConditionExpression: '(initial_status = :submitted OR initial_status = :prepared) AND initial_txid = :txid',
        ExpressionAttributeValues: { ':paid': 'paid', ':submitted': 'submitted', ':prepared': 'prepared', ':txid': transactionId, ':at': at },
      }));
    },

    async failInitial(address, message) {
      await send('dynamodb.fail_initial', new UpdateCommand({
        TableName: tableName,
        Key: { pk: `ADDR#${address}` },
        UpdateExpression: 'SET initial_status = :failed, initial_error = :message REMOVE initial_payout, initial_txid',
        ConditionExpression: 'initial_status = :reserved OR initial_status = :prepared',
        ExpressionAttributeValues: { ':failed': 'failed', ':reserved': 'reserved', ':prepared': 'prepared', ':message': String(message).slice(0, 160) },
      }));
    },

    async recordInitialUncertainty(address, diagnostic, at) {
      const httpStatus = Number.isInteger(diagnostic?.http_status) ? diagnostic.http_status : null;
      const errorCategory = typeof diagnostic?.error_category === 'string'
        ? diagnostic.error_category.slice(0, 64)
        : null;
      const expectedNonce = Number.isInteger(diagnostic?.expected_nonce) ? diagnostic.expected_nonce : null;
      const submittedNonce = Number.isInteger(diagnostic?.submitted_nonce) ? diagnostic.submitted_nonce : null;
      await send('dynamodb.record_initial_uncertainty', new UpdateCommand({
        TableName: tableName,
        Key: { pk: `ADDR#${address}` },
        UpdateExpression: 'SET initial_last_uncertain_at = :at'
          + (httpStatus == null ? '' : ', initial_last_http_status = :http_status')
          + (errorCategory == null ? '' : ', initial_last_error_category = :error_category')
          + (expectedNonce == null ? '' : ', initial_last_expected_nonce = :expected_nonce')
          + (submittedNonce == null ? '' : ', initial_last_submitted_nonce = :submitted_nonce'),
        ConditionExpression: 'initial_status = :prepared OR initial_status = :reserved',
        ExpressionAttributeValues: {
          ':prepared': 'prepared',
          ':reserved': 'reserved',
          ':at': at,
          ...(httpStatus == null ? {} : { ':http_status': httpStatus }),
          ...(errorCategory == null ? {} : { ':error_category': errorCategory }),
          ...(expectedNonce == null ? {} : { ':expected_nonce': expectedNonce }),
          ...(submittedNonce == null ? {} : { ':submitted_nonce': submittedNonce }),
        },
      }));
    },

    async acquirePayoutLock() {
      const now = Date.now();
      try {
        await send('dynamodb.acquire_payout_lock', new PutCommand({
          TableName: tableName,
          Item: { pk: 'LOCK#payout', kind: 'lock', owner: lockOwner, expires_at: now + 30_000 },
          ConditionExpression: 'attribute_not_exists(pk) OR expires_at < :now',
          ExpressionAttributeValues: { ':now': now },
        }));
        return true;
      } catch (error) {
        if (conditionalFailure(error)) return false;
        throw error;
      }
    },

    async releasePayoutLock() {
      try {
        await send('dynamodb.release_payout_lock', new DeleteCommand({
          TableName: tableName,
          Key: { pk: 'LOCK#payout' },
          ConditionExpression: '#owner = :owner',
          ExpressionAttributeNames: { '#owner': 'owner' },
          ExpressionAttributeValues: { ':owner': lockOwner },
        }));
      } catch (error) {
        if (!conditionalFailure(error)) throw error;
      }
    },

    async reserveChallenge(address, transactionId, round, at) {
      try {
        await send('dynamodb.reserve_challenge', new TransactWriteCommand({
          TransactItems: [
            {
              Put: {
                TableName: tableName,
                Item: { pk: `TX#${transactionId}`, kind: 'challenge_claim', address, round, status: 'reserved', reserved_at: at },
                ConditionExpression: 'attribute_not_exists(pk)',
              },
            },
            {
              Update: {
                TableName: tableName,
                Key: { pk: `ADDR#${address}` },
                UpdateExpression: 'SET pending_txid = :txid, pending_round = :round',
                ConditionExpression: 'initial_status = :paid AND rounds = :previous AND attribute_not_exists(pending_txid) AND (attribute_not_exists(last_round_at) OR last_round_at <= :cutoff)',
                ExpressionAttributeValues: {
                  ':txid': transactionId,
                  ':round': round,
                  ':previous': round - 1,
                  ':paid': 'paid',
                  ':cutoff': at - COOLDOWN_MS,
                },
              },
            },
          ],
        }));
        return true;
      } catch (error) {
        if (conditionalFailure(error)) return false;
        throw error;
      }
    },

    async prepareChallengePayout(address, transactionId, payout, at) {
      await send('dynamodb.prepare_challenge_payout', new UpdateCommand({
        TableName: tableName,
        Key: { pk: `TX#${transactionId}` },
        UpdateExpression: 'SET #status = :prepared, reward_payout = :payout, reward_txid = :reward, prepared_at = :at',
        ConditionExpression: 'address = :address AND (#status = :reserved OR (#status = :prepared AND reward_txid = :reward))',
        ExpressionAttributeNames: { '#status': 'status' },
        ExpressionAttributeValues: {
          ':address': address, ':reserved': 'reserved', ':prepared': 'prepared', ':payout': payout, ':reward': payout.ID, ':at': at,
        },
      }));
    },

    async completeChallenge(address, transactionId, rewardTransactionId, at) {
      const claimKey = `TX#${transactionId}`;
      const addressKey = `ADDR#${address}`;
      const current = await this.getAddress(address);
      const round = Number(current?.pending_round || 0);
      if (!Number.isInteger(round) || round < 1 || round > MAX_ROUNDS) {
        throw new Error('invalid reserved challenge round');
      }
      await send('dynamodb.complete_challenge', new TransactWriteCommand({
        TransactItems: [
          {
            Update: {
              TableName: tableName,
              Key: { pk: addressKey },
              UpdateExpression: 'SET rounds = :round, last_round_at = :at, last_reward_txid = :reward REMOVE pending_txid, pending_round',
              ConditionExpression: 'pending_txid = :txid',
              ExpressionAttributeValues: { ':round': round, ':at': at, ':reward': rewardTransactionId, ':txid': transactionId },
            },
          },
          {
            Update: {
              TableName: tableName,
              Key: { pk: claimKey },
              UpdateExpression: 'SET #status = :paid, reward_txid = :reward, completed_at = :at REMOVE reward_payout',
              ConditionExpression: '#status = :reserved OR (#status = :prepared AND reward_txid = :reward)',
              ExpressionAttributeNames: { '#status': 'status' },
              ExpressionAttributeValues: { ':paid': 'paid', ':reserved': 'reserved', ':prepared': 'prepared', ':reward': rewardTransactionId, ':at': at },
            },
          },
        ],
      }));
    },

    async failChallenge(address, transactionId, message) {
      try {
        await send('dynamodb.fail_challenge', new TransactWriteCommand({
          TransactItems: [
            {
              Delete: {
                TableName: tableName,
                Key: { pk: `TX#${transactionId}` },
                ConditionExpression: 'address = :address AND (#status = :reserved OR #status = :prepared)',
                ExpressionAttributeNames: { '#status': 'status' },
                ExpressionAttributeValues: { ':address': address, ':reserved': 'reserved', ':prepared': 'prepared' },
              },
            },
            {
              Update: {
                TableName: tableName,
                Key: { pk: `ADDR#${address}` },
                UpdateExpression: 'SET last_challenge_error = :message REMOVE pending_txid, pending_round',
                ConditionExpression: 'pending_txid = :txid',
                ExpressionAttributeValues: { ':message': String(message).slice(0, 160), ':txid': transactionId },
              },
            },
          ],
        }));
      } catch (error) {
        if (!conditionalFailure(error)) throw error;
      }
    },
  };
}

export function classifyUpstreamError(statusCode, message) {
  const text = String(message || '').toLowerCase();
  if (statusCode === 404) return 'not_found';
  if (text.includes('signature') || text.includes('identity is invalid')) return 'invalid_signature';
  if (text.includes('nonce')) return 'invalid_nonce';
  if (text.includes('insufficient balance')) return 'insufficient_balance';
  if (text.includes('fee')) return 'invalid_fee';
  if (
    text.includes('already exists')
    || text.includes('already processed')
    || text.includes('already confirmed')
  ) return 'duplicate_transaction';
  if (text.includes('transaction id')) return 'invalid_transaction_id';
  if (statusCode === 400) return 'invalid_request';
  if (statusCode === 422) return 'transaction_rejected';
  if (statusCode >= 500) return 'upstream_unavailable';
  return 'http_error';
}

export function attachUpstreamNonceMismatch(error, message) {
  const text = String(message || '');
  const match = /expected (\d+), got (\d+)/i.exec(text);
  if (!match) return error;
  error.expectedNonce = Number.parseInt(match[1], 10);
  error.submittedNonce = Number.parseInt(match[2], 10);
  return error;
}

export function parsePrometheusGauge(body, metricName) {
  const text = String(body || '');
  const pattern = new RegExp(`^${metricName}\\s+(\\d+)\\s*$`, 'm');
  const match = pattern.exec(text);
  if (!match) return null;
  const value = Number.parseInt(match[1], 10);
  return Number.isInteger(value) ? value : null;
}

export function createRpc({ seeds, fetchImpl, timeoutMs, timed }) {
  async function call(operation, method, path, body) {
    const request = {
      method,
      path,
      headers: body == null ? {} : { 'content-type': 'application/json; charset=utf-8' },
      body: body == null ? Buffer.alloc(0) : Buffer.from(JSON.stringify(body), 'utf8'),
    };
    return timed(operation, async () => {
      const result = await proxyWithFailover(request, { seeds, fetchImpl, timeoutMs });
      let payload;
      try {
        payload = JSON.parse(result.body.toString('utf8'));
      } catch {
        const error = new FaucetError(503, 'invalid response from testnet node');
        error.upstreamStatus = result.statusCode;
        error.errorCategory = 'invalid_response';
        throw error;
      }
      if (result.statusCode < 200 || result.statusCode >= 300) {
        const error = attachUpstreamNonceMismatch(new FaucetError(
          result.statusCode >= 500 ? 503 : result.statusCode,
          'testnet node rejected request',
        ), payload?.error);
        error.upstreamStatus = result.statusCode;
        error.errorCategory = classifyUpstreamError(result.statusCode, payload?.error);
        throw error;
      }
      return payload;
    });
  }

  async function probeSeed(operation, method, path, body) {
    const request = {
      method,
      path,
      headers: body == null ? {} : { 'content-type': 'application/json; charset=utf-8' },
      body: body == null ? Buffer.alloc(0) : Buffer.from(JSON.stringify(body), 'utf8'),
    };
    return timed(operation, async () => {
      let lastError;
      for (let index = 0; index < seeds.length; index++) {
        try {
          const result = await fetchOnce(seeds[index], request, { fetchImpl, timeoutMs });
          let payload;
          try {
            payload = JSON.parse(result.body.toString('utf8'));
          } catch {
            lastError = new FaucetError(503, 'invalid response from testnet node');
            lastError.upstreamStatus = result.statusCode;
            lastError.errorCategory = 'invalid_response';
            continue;
          }
          if (result.statusCode >= 200 && result.statusCode < 300) {
            return { seed_index: index, payload };
          }
          lastError = attachUpstreamNonceMismatch(new FaucetError(
            result.statusCode >= 500 ? 503 : result.statusCode,
            'testnet node rejected request',
          ), payload?.error);
          lastError.upstreamStatus = result.statusCode;
          lastError.errorCategory = classifyUpstreamError(result.statusCode, payload?.error);
        } catch (error) {
          lastError = error;
        }
      }
      throw lastError || new FaucetError(503, 'testnet node rejected request');
    });
  }

  async function probeSeedRaw(operation, method, path) {
    const request = {
      method,
      path,
      headers: {},
      body: Buffer.alloc(0),
    };
    return timed(operation, async () => {
      let lastError;
      for (let index = 0; index < seeds.length; index++) {
        try {
          const result = await fetchOnce(seeds[index], request, { fetchImpl, timeoutMs });
          if (result.statusCode >= 200 && result.statusCode < 300) {
            return { seed_index: index, bodyText: result.body.toString('utf8') };
          }
          lastError = new FaucetError(503, 'testnet node rejected request');
          lastError.upstreamStatus = result.statusCode;
          lastError.errorCategory = classifyUpstreamError(result.statusCode, result.body.toString('utf8'));
        } catch (error) {
          lastError = error;
        }
      }
      throw lastError || new FaucetError(503, 'testnet node rejected request');
    });
  }

  return {
    account(address) {
      return call('seed.account', 'GET', `/v1/accounts/${address}`);
    },
    status() {
      return call('seed.status', 'GET', '/v1/status');
    },
    transaction(transactionId) {
      return call('seed.transaction', 'GET', `/v1/transactions/${transactionId}`);
    },
    mempool(limit = 50) {
      const bounded = Number.isInteger(limit) && limit >= 1 && limit <= 500 ? limit : 50;
      return probeSeed('seed.mempool', 'GET', `/v1/mempool?limit=${bounded}`).then((result) => result.payload);
    },
    metrics() {
      return probeSeedRaw('seed.metrics', 'GET', '/metrics').then((result) => result.bodyText);
    },
    wake() {
      return call('seed.miner_wake', 'POST', '/v1/miner/wake');
    },
    async submit(transaction) {
      try {
        return await call('seed.submit_transaction', 'POST', '/v1/transactions', transaction);
      } catch (error) {
        try {
          const status = await call('seed.reconcile_transaction', 'GET', `/v1/transactions/${transaction.ID}`);
          if (status?.status === 'pending' || status?.status === 'confirmed') {
            return { accepted: true, transaction_id: transaction.ID, reconciled: true };
          }
        } catch {
          // The caller must treat an unresolved broadcast as uncertain.
        }
        const uncertain = new FaucetError(503, 'faucet payout outcome is uncertain');
        uncertain.uncertain = true;
        if (Number.isInteger(error?.upstreamStatus)) uncertain.upstreamStatus = error.upstreamStatus;
        if (typeof error?.errorCategory === 'string') uncertain.errorCategory = error.errorCategory;
        if (Number.isInteger(error?.expectedNonce)) uncertain.expectedNonce = error.expectedNonce;
        if (Number.isInteger(error?.submittedNonce)) uncertain.submittedNonce = error.submittedNonce;
        throw uncertain;
      }
    },
  };
}

export async function checkFaucetReadiness({ store, rpc, signer }) {
  await store.checkReadWrite();
  const account = await rpc.account(signer.address);
  const nonce = account?.next_nonce ?? account?.nextNonce;
  const balance = account?.balance;
  if (!Number.isSafeInteger(nonce) || nonce < 0) {
    throw new FaucetError(503, 'faucet account nonce is unavailable');
  }
  const amount = INITIAL_GRANT_SUDH * COIN;
  const fee = Math.floor((amount * 10) / 10_000);
  if (!Number.isSafeInteger(balance) || balance < amount + fee) {
    throw new FaucetError(503, 'testnet faucet needs funding');
  }
  return { ready: true };
}

function summarizeSignerMempoolTxs(mempoolBody, signerAddress) {
  const txs = Array.isArray(mempoolBody?.transactions) ? mempoolBody.transactions : [];
  return txs
    .filter((tx) => tx?.From === signerAddress)
    .map((tx) => ({ id: tx.ID, to: tx.To, nonce: tx.Nonce, amount: tx.Amount, fee: tx.Fee }));
}

export async function checkFaucetDiagnostics({ store, rpc, signer, recoveryAddress = null }) {
  const readiness = await checkFaucetReadiness({ store, rpc, signer }).then(() => true).catch(() => false);
  const account = await rpc.account(signer.address);
  let networkStatus = null;
  try {
    networkStatus = await rpc.status();
  } catch {
    networkStatus = null;
  }

  let seedMempool = { available: false, count: null, signer_txs: [] };
  try {
    const mempoolBody = await rpc.mempool(50);
    seedMempool = {
      available: true,
      count: Number.isInteger(mempoolBody?.count) ? mempoolBody.count : null,
      signer_txs: summarizeSignerMempoolTxs(mempoolBody, signer.address),
      source: 'seed_mempool',
    };
  } catch {
    try {
      const metricsBody = await rpc.metrics();
      const metricsCount = parsePrometheusGauge(metricsBody, 'sudharma_mempool_transactions');
      if (Number.isInteger(metricsCount)) {
        seedMempool = {
          available: true,
          count: metricsCount,
          signer_txs: [],
          source: 'seed_metrics',
        };
      }
    } catch {
      seedMempool = { available: false, count: null, signer_txs: [] };
    }
  }

  let preparedRecovery = null;
  if (typeof recoveryAddress === 'string' && recoveryAddress.length > 0) {
    try {
      const record = await store.getAddress(recoveryAddress);
      if (record) {
        preparedRecovery = {
          address: recoveryAddress,
          initial_status: record.initial_status ?? null,
          initial_txid: record.initial_txid ?? null,
          initial_last_error_category: record.initial_last_error_category ?? null,
          initial_last_http_status: record.initial_last_http_status ?? null,
        };
      }
    } catch {
      preparedRecovery = null;
    }
  }

  return {
    ready: readiness,
    signer: {
      address: signer.address,
      balance: account?.balance ?? null,
      confirmed_nonce: account?.confirmed_nonce ?? account?.confirmedNonce ?? null,
      next_nonce: account?.next_nonce ?? account?.nextNonce ?? null,
    },
    network: networkStatus ? {
      height: networkStatus.height ?? null,
      mempool: networkStatus.mempool ?? null,
    } : null,
    seed_mempool: seedMempool,
    prepared_recovery: preparedRecovery,
    mempool_inference: {
      network_mempool_count: networkStatus?.mempool ?? null,
      seed_mempool_endpoint_available: seedMempool.available,
      likely_prepared_nonce_blocked: !seedMempool.available
        && Number.isInteger(networkStatus?.mempool)
        && networkStatus.mempool > 0,
      chain_advancement_required: Number.isInteger(networkStatus?.mempool)
        && networkStatus.mempool > 0
        && Number.isInteger(networkStatus?.height),
    },
    testnet_only: true,
  };
}

function parseBody(request) {
  try {
    return JSON.parse(Buffer.from(request.body).toString('utf8'));
  } catch {
    throw new FaucetError(400, 'invalid faucet request JSON');
  }
}

async function loadSigner(secretId, timed) {
  const secrets = new SecretsManagerClient({});
  const result = await timed('secretsmanager.load_signer', () => secrets.send(new GetSecretValueCommand({ SecretId: secretId })));
  if (typeof result.SecretString !== 'string' || result.SecretString.length === 0) {
    throw new Error('faucet signing secret is unavailable');
  }
  let scalar;
  try {
    const parsed = JSON.parse(result.SecretString);
    scalar = parsed.private_scalar_hex;
  } catch {
    scalar = result.SecretString.trim();
  }
  return createSigner(scalar);
}

export function createRuntimeFaucetHandler({ seeds, fetchImpl = globalThis.fetch, timeoutMs, env = process.env, logger = console, now = Date.now }) {
  const tableName = env.FAUCET_TABLE_NAME;
  const secretId = env.FAUCET_SECRET_ID;
  if (!tableName || !secretId) throw new Error('faucet AWS configuration is incomplete');

  const timed = createOperationTimer({ logger, now });
  const store = createStore(tableName, timed);
  const rpc = createRpc({ seeds, fetchImpl, timeoutMs, timed });
  let servicePromise;

  async function service() {
    if (!servicePromise) {
      servicePromise = loadSigner(secretId, timed).then((signer) => ({
        signer,
        service: createFaucetService({
          store,
          rpc,
          signer,
          freshGrant: env.FAUCET_FRESH_GRANT === 'true',
        }),
      }));
    }
    return servicePromise;
  }

  return async function runtimeFaucetHandler(request) {
    const runtime = await service();
    if (request.kind === 'faucetInfo') {
      return {
        statusCode: 200,
        payload: {
          enabled: env.FAUCET_ENABLED === 'true',
          challenge_address: runtime.signer.address,
          initial_grant_sudh: INITIAL_GRANT_SUDH,
          challenge_send_sudh: CHALLENGE_SEND_SUDH,
          challenge_reward_sudh: CHALLENGE_REWARD_SUDH,
          max_rounds: MAX_ROUNDS,
          cooldown_hours: COOLDOWN_MS / (60 * 60 * 1000),
          testnet_only: true,
        },
      };
    }
    if (request.kind === 'faucetHealth') {
      return {
        statusCode: 200,
        payload: await checkFaucetReadiness({ store, rpc, signer: runtime.signer }),
      };
    }
    if (request.kind === 'faucetDiagnostics') {
      return {
        statusCode: 200,
        payload: await checkFaucetDiagnostics({
          store,
          rpc,
          signer: runtime.signer,
          recoveryAddress: env.FAUCET_RECOVERY_ADDRESS || null,
        }),
      };
    }

    const body = parseBody(request);
    if (request.kind === 'faucetInitial') {
      return { statusCode: 202, payload: await runtime.service.requestInitial(body?.address) };
    }
    if (request.kind === 'faucetChallenge') {
      return { statusCode: 202, payload: await runtime.service.claimChallenge(body?.address, body?.transaction_id) };
    }
    throw new FaucetError(404, 'faucet route not found');
  };
}
