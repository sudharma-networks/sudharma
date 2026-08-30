import { createHash } from 'node:crypto';
import { DynamoDBClient } from '@aws-sdk/client-dynamodb';
import {
  DynamoDBDocumentClient,
  GetCommand as AwsGetCommand,
  TransactWriteCommand as AwsTransactWriteCommand,
} from '@aws-sdk/lib-dynamodb';

const VISITOR_ID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const COUNTER_KEY = 'WEBVISITOR#COUNTER';
const MARKER_TTL_SECONDS = 3 * 24 * 60 * 60;

export class VisitorCounterError extends Error {
  constructor(statusCode, message) {
    super(message);
    this.name = 'VisitorCounterError';
    this.statusCode = statusCode;
  }
}

function utcDay(timestamp) {
  return new Date(timestamp).toISOString().slice(0, 10);
}

function visitorHash(visitorId) {
  return createHash('sha256').update(visitorId.toLowerCase(), 'utf8').digest('hex');
}

function parseVisitorId(body) {
  let payload;
  try {
    payload = JSON.parse(body.toString('utf8'));
  } catch {
    throw new VisitorCounterError(400, 'invalid JSON request body');
  }
  if (!VISITOR_ID.test(payload?.visitorId || '')) {
    throw new VisitorCounterError(400, 'invalid visitor identifier');
  }
  return payload.visitorId.toLowerCase();
}

function duplicateMarker(error) {
  return error?.name === 'TransactionCanceledException'
    && error?.CancellationReasons?.[0]?.Code === 'ConditionalCheckFailed';
}

export function createVisitorHandler(options = {}) {
  const tableName = options.tableName;
  if (!tableName) throw new Error('visitor counter table name is required');

  const client = options.client || DynamoDBDocumentClient.from(new DynamoDBClient({}), {
    marshallOptions: { removeUndefinedValues: true },
  });
  const commands = options.commands || { GetCommand: AwsGetCommand, TransactWriteCommand: AwsTransactWriteCommand };
  const now = options.now || Date.now;

  async function readTotal() {
    const result = await client.send(new commands.GetCommand({
      TableName: tableName,
      Key: { pk: COUNTER_KEY },
      ConsistentRead: true,
    }));
    const total = Number(result?.Item?.total || 0);
    return Number.isSafeInteger(total) && total >= 0 ? total : 0;
  }

  return async function visitorHandler(request) {
    if (request?.kind === 'websiteVisitorsRead') {
      return { statusCode: 200, payload: { total: await readTotal() } };
    }
    if (request?.kind !== 'websiteVisitorsRecord') {
      throw new VisitorCounterError(404, 'visitor route not found');
    }

    const timestamp = now();
    const visitorId = parseVisitorId(request.body || Buffer.alloc(0));
    const day = utcDay(timestamp);
    const markerKey = `WEBVISIT#${day}#${visitorHash(visitorId)}`;
    const expiresAt = Math.floor(timestamp / 1000) + MARKER_TTL_SECONDS;

    try {
      await client.send(new commands.TransactWriteCommand({
        TransactItems: [
          {
            Put: {
              TableName: tableName,
              Item: {
                pk: markerKey,
                kind: 'website_visit',
                day,
                expires_at: expiresAt,
              },
              ConditionExpression: 'attribute_not_exists(pk)',
            },
          },
          {
            Update: {
              TableName: tableName,
              Key: { pk: COUNTER_KEY },
              UpdateExpression: 'ADD #total :one',
              ExpressionAttributeNames: { '#total': 'total' },
              ExpressionAttributeValues: { ':one': 1 },
            },
          },
        ],
      }));
    } catch (error) {
      if (!duplicateMarker(error)) throw error;
    }

    return { statusCode: 200, payload: { total: await readTotal() } };
  };
}
