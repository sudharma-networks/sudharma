import test from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";

const root = path.resolve("deployment/website/visitor-counter");
const workflowPath = path.resolve(".github/workflows/provision-website-visitor-counter.yml");

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

test("GitHub Actions deploys the visitor counter through AWS OIDC and commits only public endpoint config", () => {
  assert.equal(existsSync(workflowPath), true, "visitor counter provisioning workflow must exist");
  const text = readFileSync(workflowPath, "utf8");
  assert.match(text, /id-token:\s*write/);
  assert.match(text, /aws-actions\/configure-aws-credentials@v4/);
  assert.match(text, /Sudharma-GitHub-Actions-Testnet/);
  assert.match(text, /deployment\/website\/visitor-counter\/provision\.sh/);
  assert.match(text, /web\/public\/data\/visitor-counter\.json/);
});
