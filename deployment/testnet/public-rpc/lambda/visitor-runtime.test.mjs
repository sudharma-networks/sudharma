import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const modulePath = path.join(here, 'visitor-runtime.mjs');

class GetCommand { constructor(input) { this.input = input; this.kind = 'get'; } }
class TransactWriteCommand { constructor(input) { this.input = input; this.kind = 'transact'; } }
const commands = { GetCommand, TransactWriteCommand };

async function loadFactory() {
  assert.equal(existsSync(modulePath), true, 'visitor runtime must exist');
  return (await import(pathToFileURL(modulePath).href)).createVisitorHandler;
}

test('GET returns the public total without writing visitor state', async () => {
  const createVisitorHandler = await loadFactory();
  const sent = [];
  const handler = createVisitorHandler({
    tableName: 'Sudharma-Testnet-Faucet',
    commands,
    client: { async send(command) { sent.push(command); return { Item: { total: 41 } }; } },
    now: () => Date.parse('2026-08-29T12:00:00Z'),
  });

  const result = await handler({ kind: 'websiteVisitorsRead', body: Buffer.alloc(0) });
  assert.deepEqual(result, { statusCode: 200, payload: { total: 41 } });
  assert.equal(sent.length, 1);
  assert.equal(sent[0].kind, 'get');
  assert.equal(sent[0].input.Key.pk, 'WEBVISITOR#COUNTER');
});

test('POST atomically records one hashed browser marker for the UTC day and increments total', async () => {
  const createVisitorHandler = await loadFactory();
  const sent = [];
  const handler = createVisitorHandler({
    tableName: 'Sudharma-Testnet-Faucet',
    commands,
    client: {
      async send(command) {
        sent.push(command);
        if (command.kind === 'get') return { Item: { total: 42 } };
        return {};
      },
    },
    now: () => Date.parse('2026-08-29T12:00:00Z'),
  });

  const visitorId = '11111111-2222-4333-8444-555555555555';
  const result = await handler({
    kind: 'websiteVisitorsRecord',
    body: Buffer.from(JSON.stringify({ visitorId }), 'utf8'),
    headers: { 'content-type': 'application/json' },
  });

  assert.deepEqual(result, { statusCode: 200, payload: { total: 42 } });
  assert.equal(sent[0].kind, 'transact');
  const marker = sent[0].input.TransactItems[0].Put.Item;
  assert.match(marker.pk, /^WEBVISIT#2026-08-29#[0-9a-f]{64}$/);
  assert.equal(marker.pk.includes(visitorId), false);
  assert.equal(sent[0].input.TransactItems[0].Put.ConditionExpression, 'attribute_not_exists(pk)');
  assert.equal(sent[0].input.TransactItems[1].Update.Key.pk, 'WEBVISITOR#COUNTER');
  assert.equal(sent[0].input.TransactItems[1].Update.UpdateExpression, 'ADD #total :one');
});

test('duplicate daily marker returns current total without surfacing a conflict', async () => {
  const createVisitorHandler = await loadFactory();
  const duplicate = Object.assign(new Error('duplicate'), {
    name: 'TransactionCanceledException',
    CancellationReasons: [{ Code: 'ConditionalCheckFailed' }, { Code: 'None' }],
  });
  let calls = 0;
  const handler = createVisitorHandler({
    tableName: 'Sudharma-Testnet-Faucet',
    commands,
    client: {
      async send(command) {
        calls += 1;
        if (command.kind === 'transact') throw duplicate;
        return { Item: { total: 42 } };
      },
    },
    now: () => Date.parse('2026-08-29T12:00:00Z'),
  });

  const result = await handler({
    kind: 'websiteVisitorsRecord',
    body: Buffer.from(JSON.stringify({ visitorId: '11111111-2222-4333-8444-555555555555' })),
    headers: { 'content-type': 'application/json' },
  });
  assert.equal(result.payload.total, 42);
  assert.equal(calls, 2);
});

test('POST rejects malformed browser identifiers without touching DynamoDB', async () => {
  const createVisitorHandler = await loadFactory();
  let calls = 0;
  const handler = createVisitorHandler({
    tableName: 'Sudharma-Testnet-Faucet',
    commands,
    client: { async send() { calls += 1; return {}; } },
  });

  await assert.rejects(
    () => handler({ kind: 'websiteVisitorsRecord', body: Buffer.from('{"visitorId":"bad"}') }),
    (error) => error?.statusCode === 400,
  );
  assert.equal(calls, 0);
});
