import test from "node:test";
import assert from "node:assert/strict";
import { existsSync } from "node:fs";
import { pathToFileURL } from "node:url";
import path from "node:path";

const storePath = path.resolve("deployment/website/visitor-counter/store.mjs");

class GetItemCommand { constructor(input) { this.input = input; this.kind = "get"; } }
class TransactWriteItemsCommand { constructor(input) { this.input = input; this.kind = "transact"; } }
const commands = { GetItemCommand, TransactWriteItemsCommand };

test("DynamoDB store atomically adds a marker and increments the public total", async () => {
  assert.equal(existsSync(storePath), true, "visitor counter DynamoDB store must exist");
  const { createDynamoStore } = await import(pathToFileURL(storePath).href);
  const sent = [];
  const client = {
    async send(command) {
      sent.push(command);
      if (command.kind === "get") return { Item: { total: { N: "12" } } };
      return {};
    }
  };
  const store = createDynamoStore({ client, commands, tableName: "Visitors" });
  const total = await store.recordVisit({ key: "VISIT#2026-08-29#abc", expiresAt: 12345 });

  assert.equal(total, 12);
  assert.equal(sent[0].kind, "transact");
  assert.equal(sent[0].input.TransactItems[0].Put.ConditionExpression, "attribute_not_exists(pk)");
  assert.equal(sent[0].input.TransactItems[1].Update.UpdateExpression, "ADD #total :one");
});

test("DynamoDB store treats only the marker conditional failure as a duplicate", async () => {
  assert.equal(existsSync(storePath), true, "visitor counter DynamoDB store must exist");
  const { createDynamoStore } = await import(pathToFileURL(storePath).href);
  const duplicate = Object.assign(new Error("duplicate"), {
    name: "TransactionCanceledException",
    CancellationReasons: [{ Code: "ConditionalCheckFailed" }, { Code: "None" }]
  });
  let calls = 0;
  const store = createDynamoStore({
    client: {
      async send(command) {
        calls += 1;
        if (command.kind === "transact") throw duplicate;
        return { Item: { total: { N: "9" } } };
      }
    },
    commands,
    tableName: "Visitors"
  });

  assert.equal(await store.recordVisit({ key: "VISIT#x", expiresAt: 123 }), 9);
  assert.equal(calls, 2);
});

test("DynamoDB store does not hide unrelated transaction failures", async () => {
  assert.equal(existsSync(storePath), true, "visitor counter DynamoDB store must exist");
  const { createDynamoStore } = await import(pathToFileURL(storePath).href);
  const failure = Object.assign(new Error("capacity failure"), {
    name: "TransactionCanceledException",
    CancellationReasons: [{ Code: "None" }, { Code: "ProvisionedThroughputExceeded" }]
  });
  const store = createDynamoStore({
    client: { async send() { throw failure; } },
    commands,
    tableName: "Visitors"
  });

  await assert.rejects(() => store.recordVisit({ key: "VISIT#x", expiresAt: 123 }), /capacity failure/);
});
