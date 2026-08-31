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

export function createOperationTimer({ logger = console, now = Date.now, timeoutMs = 6_000 } = {}) {
  return async function timeOperation(operation, action) {
    const started = now();
    let timeoutId;
    try {
      const result = await Promise.race([
        action(),
        new Promise((_, reject) => {
          timeoutId = setTimeout(
            () => reject(new FaucetError(503, 'faucet dependency timed out')),
            timeoutMs,
          );
        }),
      ]);
      logger.info({ event: 'faucet_dependency', operation, outcome: 'success', latency_ms: now() - started });
      return result;
    } catch (error) {
      logger.info({
        event: 'faucet_dependency',
        operation,
        outcome: 'error',
        error_name: String(error?.name || 'Error'),
        latency_ms: now() - started,
      });
      throw error;
    } finally {
      clearTimeout(timeoutId);
    }
  };
}

function createStore(tableName, timed) {
  const client = DynamoDBDocumentClient.from(new DynamoDBClient({}), {
    marshallOptions: { removeUndefinedValues: true },
  });
  const lockOwner = randomUUID();
  const send = (operation, command) => timed(operation, () => client.send(command));

  return {
    async getAddress(address) {
      const result = await send('dynamodb.get_address', new GetCommand({ TableName: tableName, Key: { pk: `ADDR#${address}` } }));
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

    async markInitialSubmitted(address, transactionId, at) {
      await send('dynamodb.mark_initial_submitted', new UpdateCommand({
        TableName: tableName,
        Key: { pk: `ADDR#${address}` },
        UpdateExpression: 'SET initial_status = :submitted, initial_txid = :txid, initial_submitted_at = :at',
        ConditionExpression: 'initial_status = :reserved OR (initial_status = :paid AND initial_txid = :txid)',
        ExpressionAttributeValues: {
          ':submitted': 'submitted', ':reserved': 'reserved', ':paid': 'paid', ':txid': transactionId, ':at': at,
        },
      }));
    },

    async completeInitial(address, transactionId, at) {
      await send('dynamodb.complete_initial', new UpdateCommand({
        TableName: tableName,
        Key: { pk: `ADDR#${address}` },
        UpdateExpression: 'SET initial_status = :paid, initial_txid = :txid, initial_paid_at = :at',
        ConditionExpression: 'initial_status = :submitted AND initial_txid = :txid',
        ExpressionAttributeValues: { ':paid': 'paid', ':submitted': 'submitted', ':txid': transactionId, ':at': at },
      }));
    },

    async failInitial(address, message) {
      await send('dynamodb.fail_initial', new UpdateCommand({
        TableName: tableName,
        Key: { pk: `ADDR#${address}` },
        UpdateExpression: 'SET initial_status = :failed, initial_error = :message',
        ConditionExpression: 'initial_status = :reserved',
        ExpressionAttributeValues: { ':failed': 'failed', ':reserved': 'reserved', ':message': String(message).slice(0, 160) },
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
              UpdateExpression: 'SET #status = :paid, reward_txid = :reward, completed_at = :at',
              ConditionExpression: '#status = :reserved',
              ExpressionAttributeNames: { '#status': 'status' },
              ExpressionAttributeValues: { ':paid': 'paid', ':reserved': 'reserved', ':reward': rewardTransactionId, ':at': at },
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
                ConditionExpression: 'address = :address AND #status = :reserved',
                ExpressionAttributeNames: { '#status': 'status' },
                ExpressionAttributeValues: { ':address': address, ':reserved': 'reserved' },
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

function createRpc({ seeds, fetchImpl, timeoutMs, timed }) {
  async function call(operation, method, path, body) {
    const request = {
      method,
      path,
      headers: body == null ? {} : { 'content-type': 'application/json; charset=utf-8' },
      body: body == null ? Buffer.alloc(0) : Buffer.from(JSON.stringify(body), 'utf8'),
    };
    const result = await timed(operation, () => proxyWithFailover(request, { seeds, fetchImpl, timeoutMs }));
    let payload;
    try {
      payload = JSON.parse(result.body.toString('utf8'));
    } catch {
      throw new FaucetError(503, 'invalid response from testnet node');
    }
    if (result.statusCode < 200 || result.statusCode >= 300) {
      throw new FaucetError(result.statusCode >= 500 ? 503 : result.statusCode, payload?.error || 'testnet node rejected request');
    }
    return payload;
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
        throw uncertain;
      }
    },
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

  const dependencyTimeoutMs = Number.parseInt(env.FAUCET_DEPENDENCY_TIMEOUT_MS || '6000', 10);
  const timed = createOperationTimer({
    logger,
    now,
    timeoutMs:
      Number.isInteger(dependencyTimeoutMs) && dependencyTimeoutMs >= 100 && dependencyTimeoutMs <= 15_000
        ? dependencyTimeoutMs
        : 6_000,
  });
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
          enabled: true,
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
