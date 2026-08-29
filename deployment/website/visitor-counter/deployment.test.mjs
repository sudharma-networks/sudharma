import test from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";

const root = path.resolve("deployment/website/visitor-counter");

function source(name) {
  const file = path.join(root, name);
  assert.equal(existsSync(file), true, `${name} must exist`);
  return readFileSync(file, "utf8");
}

test("Lambda entrypoint wires the HTTP handler to DynamoDB persistence", () => {
  const text = source("index.mjs");
  assert.match(text, /createHandler/);
  assert.match(text, /createDynamoStore/);
  assert.match(text, /TABLE_NAME/);
  assert.match(text, /export const handler/);
});

test("AWS provisioner creates persistent visitor storage and a public HTTP endpoint", () => {
  const text = source("provision.sh");
  assert.match(text, /dynamodb create-table/);
  assert.match(text, /lambda create-function/);
  assert.match(text, /apigatewayv2 create-api/);
  assert.match(text, /visitor-counter\.json/);
});
