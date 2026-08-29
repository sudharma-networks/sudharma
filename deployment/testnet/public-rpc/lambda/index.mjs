import { normalizeEvent, RequestError } from './router.mjs';
import { proxyWithFailover, UpstreamUnavailableError } from './upstream.mjs';

const DEFAULT_SEEDS = [
  'http://172.31.10.171:29100',
  'http://172.31.32.195:29100',
];

function jsonResponse(statusCode, payload, extraHeaders = {}) {
  return {
    statusCode,
    headers: {
      'cache-control': 'no-store',
      'content-type': 'application/json; charset=utf-8',
      'x-content-type-options': 'nosniff',
      ...extraHeaders,
    },
    body: JSON.stringify(payload),
    isBase64Encoded: false,
  };
}

function visitorJsonResponse(statusCode, payload) {
  return jsonResponse(statusCode, payload, { 'access-control-allow-origin': '*' });
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

function isFaucetRoute(kind) {
  return kind === 'faucetInfo' || kind === 'faucetInitial' || kind === 'faucetChallenge';
}

function isVisitorRoute(kind) {
  return kind === 'websiteVisitorsRead' || kind === 'websiteVisitorsRecord';
}

export function createHandler(options = {}) {
  const seeds = options.seeds || DEFAULT_SEEDS;
  const fetchImpl = options.fetchImpl || globalThis.fetch;
  const timeoutMs = options.timeoutMs;
  const logger = options.logger || console;
  const faucetHandler = options.faucetHandler || null;
  const visitorHandler = options.visitorHandler || null;

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

    if (isVisitorRoute(request.kind)) {
      if (typeof visitorHandler !== 'function') {
        return visitorJsonResponse(503, { error: 'website visitor counter is temporarily unavailable' });
      }
      try {
        const result = await visitorHandler(request, { context });
        const statusCode = Number.isInteger(result?.statusCode) ? result.statusCode : 200;
        const payload = result?.payload ?? result ?? { total: 0 };
        safeLog(logger, 'info', {
          event: 'website_visitor_response',
          route: request.kind,
          status_code: statusCode,
          latency_ms: Date.now() - started,
          request_id: context?.awsRequestId || null,
        });
        return visitorJsonResponse(statusCode, payload);
      } catch (error) {
        const statusCode = Number.isInteger(error?.statusCode) ? error.statusCode : 500;
        safeLog(logger, statusCode >= 500 ? 'error' : 'warn', {
          event: 'website_visitor_error',
          route: request.kind,
          status_code: statusCode,
          latency_ms: Date.now() - started,
          request_id: context?.awsRequestId || null,
        });
        return visitorJsonResponse(statusCode, {
          error: statusCode >= 500
            ? 'website visitor counter is temporarily unavailable'
            : String(error?.message || 'visitor request rejected'),
        });
      }
    }

    if (isFaucetRoute(request.kind)) {
      if (typeof faucetHandler !== 'function') {
        safeLog(logger, 'warn', {
          event: 'wallet_faucet_unavailable',
          route: request.kind,
          request_id: context?.awsRequestId || null,
        });
        return jsonResponse(503, { error: 'testnet faucet is not configured yet' });
      }
      try {
        const result = await faucetHandler(request, { context, seeds, fetchImpl, timeoutMs });
        const statusCode = Number.isInteger(result?.statusCode) ? result.statusCode : 200;
        const payload = result?.payload ?? result ?? { status: 'ok' };
        safeLog(logger, 'info', {
          event: 'wallet_faucet_response',
          route: request.kind,
          status_code: statusCode,
          latency_ms: Date.now() - started,
          request_id: context?.awsRequestId || null,
        });
        return jsonResponse(statusCode, payload);
      } catch (error) {
        const statusCode = Number.isInteger(error?.statusCode) ? error.statusCode : 500;
        safeLog(logger, statusCode >= 500 ? 'error' : 'warn', {
          event: 'wallet_faucet_error',
          route: request.kind,
          status_code: statusCode,
          latency_ms: Date.now() - started,
          request_id: context?.awsRequestId || null,
        });
        return jsonResponse(statusCode, {
          error: statusCode >= 500
            ? 'testnet faucet is temporarily unavailable'
            : String(error?.message || 'faucet request rejected'),
        });
      }
    }

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
const runtimeTimeoutMs = Number.isFinite(configuredTimeoutMs) ? configuredTimeoutMs : 3500;

let productionFaucetHandler = null;
if (process.env.FAUCET_ENABLED === 'true') {
  let runtimeHandlerPromise;
  productionFaucetHandler = async (request) => {
    if (!runtimeHandlerPromise) {
      runtimeHandlerPromise = import('./faucet-runtime.mjs')
        .then((module) => module.createRuntimeFaucetHandler({
          seeds: configuredSeeds,
          timeoutMs: runtimeTimeoutMs,
        }));
    }
    const runtimeHandler = await runtimeHandlerPromise;
    return runtimeHandler(request);
  };
}

let productionVisitorHandler = null;
if (process.env.FAUCET_TABLE_NAME) {
  let visitorHandlerPromise;
  productionVisitorHandler = async (request) => {
    if (!visitorHandlerPromise) {
      visitorHandlerPromise = import('./visitor-runtime.mjs')
        .then((module) => module.createVisitorHandler({ tableName: process.env.FAUCET_TABLE_NAME }));
    }
    const visitorHandler = await visitorHandlerPromise;
    return visitorHandler(request);
  };
}

export const handler = createHandler({
  seeds: configuredSeeds,
  timeoutMs: runtimeTimeoutMs,
  faucetHandler: productionFaucetHandler,
  visitorHandler: productionVisitorHandler,
});
