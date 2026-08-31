import test from "node:test";
import assert from "node:assert/strict";
import { existsSync } from "node:fs";
import { pathToFileURL } from "node:url";
import path from "node:path";

const logicPath = path.resolve("deployment/website/visitor-counter/logic.mjs");

function event(method, body) {
  return {
    requestContext: { http: { method } },
    body: body === undefined ? undefined : JSON.stringify(body)
  };
}

test("records at most one visit per browser identifier each day", async () => {
  assert.equal(existsSync(logicPath), true, "visitor counter backend logic must exist");
  const { createHandler } = await import(pathToFileURL(logicPath).href);

  const seen = new Set();
  let total = 0;
  const store = {
    async getTotal() { return total; },
    async recordVisit(marker) {
      if (!seen.has(marker.key)) {
        seen.add(marker.key);
        total += 1;
      }
      return total;
    }
  };

  const handler = createHandler({ store, now: () => new Date("2026-08-29T12:00:00Z") });
  const visitor = "4d2b6251-6ef0-45d4-b7d6-3d14426c70ca";

  const first = await handler(event("POST", { visitorId: visitor }));
  const repeat = await handler(event("POST", { visitorId: visitor }));
  const secondVisitor = await handler(event("POST", { visitorId: "aa8b099d-d3ff-44fc-a303-58db3d1d8dce" }));

  assert.equal(JSON.parse(first.body).total, 1);
  assert.equal(JSON.parse(repeat.body).total, 1);
  assert.equal(JSON.parse(secondVisitor.body).total, 2);
});

test("GET reads the public total without recording a visit", async () => {
  assert.equal(existsSync(logicPath), true, "visitor counter backend logic must exist");
  const { createHandler } = await import(pathToFileURL(logicPath).href);
  let records = 0;
  const handler = createHandler({
    store: {
      async getTotal() { return 7; },
      async recordVisit() { records += 1; return 8; }
    },
    now: () => new Date("2026-08-29T12:00:00Z")
  });

  const response = await handler(event("GET"));
  assert.equal(response.statusCode, 200);
  assert.equal(JSON.parse(response.body).total, 7);
  assert.equal(records, 0);
});

test("rejects malformed visitor identifiers", async () => {
  assert.equal(existsSync(logicPath), true, "visitor counter backend logic must exist");
  const { createHandler } = await import(pathToFileURL(logicPath).href);
  const handler = createHandler({
    store: { async getTotal() { return 0; }, async recordVisit() { return 1; } },
    now: () => new Date("2026-08-29T12:00:00Z")
  });

  const response = await handler(event("POST", { visitorId: "not-a-valid-browser-id" }));
  assert.equal(response.statusCode, 400);
});
