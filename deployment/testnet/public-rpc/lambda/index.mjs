import { normalizeEvent, RequestError } from './router.mjs';
import { proxyWithFailover, UpstreamUnavailableError } from './upstream.mjs';

const DEFAULT_SEEDS = [
  'http://172.31.10.171:29100',
  'http://172.31.32.195:29100',
];

function jsonResponse(statusCode, payload) {
  return {
    statusCode,
    headers: {
      'cache-control': 'no-store',
      'content-type': 'application/json; charset=utf-8',
      'x-content-type-options': 'nosniff',
    },
    body: JSON.stringify(payload),
    isBase64Encoded: false,
  };
}

function gatewayResponse(result) {
  const contentType = result.headers['content-type'] || 'application/json; charset=utf-8';
  const textual = /^(application\/json|text\/)/i.test(contentType);
  return {
    statusCode: result.statusCode,
    headers: {
      ...result.headers,
      'cache-control': 'no-store',
      'x-content-type-options': 'nosniff',
    },
    body: textual ? result.body.toString('utf8') : result.body.toString('base64'),
    isBase64Encoded: !textual,
  };
}

function safeLog(logger, level, record) {
  const fn = logger?.[level];
  if (typeof fn === 'function') fn.call(logger, record);
}

export function createHandler(options = {}) {
  const seeds = options.seeds || DEFAULT_SEEDS;
  const fetchImpl = options.fetchImpl || globalThis.fetch;
  const timeoutMs = options.timeoutMs;
  const logger = options.logger || console;

  return async function walletProxyHandler(event, context = {}) {
    const started = Date.now();
    let request;
    try {
      request = normalizeEvent(event);
    } catch (error) {
      const statusCode = error instanceof RequestError ? error.statusCode : 400;
      safeLog(logger, 'warn', {
        event: 'wallet_proxy_rejected',
        status_code: statusCode,
        method: event?.requestContext?.http?.method || 'unknown',
        path: event?.rawPath || 'unknown',
      });
      return jsonResponse(statusCode, { error: error instanceof RequestError ? error.message : 'invalid request' });
    }

    safeLog(logger, 'info', {
      event: 'wallet_proxy_request',
      method: request.method,
      path: request.path,
      route: request.kind,
      request_id: context?.awsRequestId || null,
    });

    try {
      const result = await proxyWithFailover(request, { seeds, fetchImpl, timeoutMs });
      safeLog(logger, 'info', {
        event: 'wallet_proxy_response',
        method: request.method,
        path: request.path,
        route: request.kind,
        status_code: result.statusCode,
        latency_ms: Date.now() - started,
        request_id: context?.awsRequestId || null,
      });
      return gatewayResponse(result);
    } catch (error) {
      const unavailable = error instanceof UpstreamUnavailableError;
      safeLog(logger, unavailable ? 'warn' : 'error', {
        event: unavailable ? 'wallet_proxy_upstream_unavailable' : 'wallet_proxy_internal_error',
        method: request.method,
        path: request.path,
        route: request.kind,
        latency_ms: Date.now() - started,
        request_id: context?.awsRequestId || null,
      });
      if (unavailable) {
        return jsonResponse(503, {
          error: request.kind === 'submitTransaction'
            ? 'transaction outcome is uncertain because wallet service is unavailable; check transaction status before retrying'
            : 'wallet service is temporarily unavailable',
        });
      }
      return jsonResponse(500, { error: 'internal wallet proxy error' });
    }
  };
}

const configuredSeeds = [
  process.env.SEED_1_URL || DEFAULT_SEEDS[0],
  process.env.SEED_2_URL || DEFAULT_SEEDS[1],
];
const configuredTimeoutMs = Number.parseInt(process.env.UPSTREAM_TIMEOUT_MS || '3500', 10);

export const handler = createHandler({
  seeds: configuredSeeds,
  timeoutMs: Number.isFinite(configuredTimeoutMs) ? configuredTimeoutMs : 3500,
});
