import { createHash } from "node:crypto";

const VISITOR_ID = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const MARKER_TTL_SECONDS = 3 * 24 * 60 * 60;

function json(statusCode, body) {
  return {
    statusCode,
    headers: {
      "content-type": "application/json; charset=utf-8",
      "cache-control": "no-store"
    },
    body: JSON.stringify(body)
  };
}

function methodOf(event) {
  return event?.requestContext?.http?.method || event?.httpMethod || "GET";
}

function parseBody(event) {
  if (!event?.body) return {};
  try {
    return JSON.parse(event.body);
  } catch {
    return null;
  }
}

export function visitMarker(visitorId, date) {
  const day = date.toISOString().slice(0, 10);
  const digest = createHash("sha256").update(`${day}:${visitorId}`).digest("hex");
  return {
    key: `VISIT#${day}#${digest}`,
    expiresAt: Math.floor(date.getTime() / 1000) + MARKER_TTL_SECONDS
  };
}

export function createHandler({ store, now = () => new Date() }) {
  if (!store?.getTotal || !store?.recordVisit) throw new TypeError("visitor counter store is required");

  return async function handler(event) {
    const method = methodOf(event);

    if (method === "GET") {
      return json(200, { total: await store.getTotal() });
    }

    if (method === "OPTIONS") {
      return { statusCode: 204, headers: { "cache-control": "no-store" }, body: "" };
    }

    if (method !== "POST") return json(405, { error: "method_not_allowed" });

    const body = parseBody(event);
    if (!body || typeof body.visitorId !== "string" || !VISITOR_ID.test(body.visitorId)) {
      return json(400, { error: "invalid_visitor_id" });
    }

    const marker = visitMarker(body.visitorId.toLowerCase(), now());
    const total = await store.recordVisit(marker);
    return json(200, { total });
  };
}
