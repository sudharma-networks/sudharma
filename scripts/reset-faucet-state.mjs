#!/usr/bin/env node
/**
 * Delete faucet grant rows (ADDR#, TX#, LOCK#, HEALTH#) for a testnet fresh start.
 * Preserves website visitor keys (WEBVISITOR#, WEBVISIT#) in the shared table.
 * Does not touch Secrets Manager or on-chain state.
 */
import { DynamoDBClient } from '@aws-sdk/client-dynamodb';
import {
  BatchWriteCommand,
  DynamoDBDocumentClient,
  ScanCommand,
} from '@aws-sdk/lib-dynamodb';

const tableName = process.env.FAUCET_TABLE_NAME || 'Sudharma-Testnet-Faucet';
const dryRun = process.env.DRY_RUN === 'true';

function shouldDeleteKey(pk) {
  return pk.startsWith('ADDR#')
    || pk.startsWith('TX#')
    || pk === 'LOCK#payout'
    || pk.startsWith('HEALTH#');
}

function chunk(items, size) {
  const out = [];
  for (let i = 0; i < items.length; i += size) out.push(items.slice(i, i + size));
  return out;
}

async function scanAllKeys(client) {
  const keys = [];
  let lastKey;
  do {
    const page = await client.send(new ScanCommand({
      TableName: tableName,
      ProjectionExpression: 'pk',
      ExclusiveStartKey: lastKey,
    }));
    for (const item of page.Items || []) {
      if (typeof item?.pk === 'string') keys.push(item.pk);
    }
    lastKey = page.LastEvaluatedKey;
  } while (lastKey);
  return keys;
}

async function deleteKeys(client, keys) {
  let deleted = 0;
  for (const batch of chunk(keys, 25)) {
    if (dryRun) {
      deleted += batch.length;
      continue;
    }
    await client.send(new BatchWriteCommand({
      RequestItems: {
        [tableName]: batch.map((pk) => ({ DeleteRequest: { Key: { pk } } })),
      },
    }));
    deleted += batch.length;
  }
  return deleted;
}

async function main() {
  const client = DynamoDBDocumentClient.from(new DynamoDBClient({}), {
    marshallOptions: { removeUndefinedValues: true },
  });
  const keys = await scanAllKeys(client);
  const toDelete = keys.filter(shouldDeleteKey);
  const preserved = keys.filter((pk) => !shouldDeleteKey(pk));
  const summary = {
    table: tableName,
    dry_run: dryRun,
    keys_found: keys.length,
    addr_rows: keys.filter((k) => k.startsWith('ADDR#')).length,
    tx_rows: keys.filter((k) => k.startsWith('TX#')).length,
    preserved_rows: preserved.length,
    preserved_keys: preserved.slice(0, 10),
  };
  summary.deleted = await deleteKeys(client, toDelete);
  console.log(JSON.stringify({ faucet_reset: dryRun ? 'dry_run' : 'ok', ...summary }, null, 2));
}

main().catch((error) => {
  console.error(String(error?.message || error));
  process.exit(1);
});
