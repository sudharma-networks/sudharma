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
import { proxyWithFailover } from './upstream.mjs';

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
        const error = new FaucetError(
          result.statusCode >= 500 ? 503 : result.statusCode,
          'testnet node rejected request',
        );
        error.upstreamStatus = result.statusCode;
        error.errorCategory = classifyUpstreamError(result.statusCode, payload?.error);
        throw error;
      }
      return payload;
    });
  }

  return {
    account(address) {
      return call('seed.account', 'GET', `/v1/accounts/${address}`);
    },
    transaction(transactionId) {
      return call('seed.transaction', 'GET', `/v1/transactions/${transactionId}`);
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
        service: createFaucetService({ store, rpc, signer }),
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
