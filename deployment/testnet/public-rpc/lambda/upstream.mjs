const DEFAULT_TIMEOUT_MS = 3500;
const MAX_RESPONSE_BYTES = 4 * 1024 * 1024;

export class UpstreamUnavailableError extends Error {
  constructor(message = 'wallet service is temporarily unavailable') {
    super(message);
    this.name = 'UpstreamUnavailableError';
  }
}

function isRetryableStatus(status) {
  return status >= 500 && status <= 599;
}

function safeResponseHeaders(response) {
  const headers = { 'cache-control': 'no-store' };
  const contentType = response.headers.get('content-type');
  if (contentType) headers['content-type'] = contentType.slice(0, 200);
  return headers;
}

async function readBounded(response, maxBytes = MAX_RESPONSE_BYTES) {
  const contentLength = Number.parseInt(response.headers.get('content-length') || '', 10);
  if (Number.isFinite(contentLength) && contentLength > maxBytes) {
    throw new UpstreamUnavailableError('upstream response too large');
  }
  const buffer = Buffer.from(await response.arrayBuffer());
  if (buffer.length > maxBytes) {
    throw new UpstreamUnavailableError('upstream response too large');
  }
  return buffer;
}

async function fetchOnce(seed, request, { fetchImpl, timeoutMs }) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const init = {
      method: request.method,
      redirect: 'error',
      signal: controller.signal,
      headers: {},
    };
    if (request.headers?.['content-type']) {
      init.headers['content-type'] = request.headers['content-type'];
    }
    if (request.method === 'POST') {
      init.body = Buffer.from(request.body);
    }
    const response = await fetchImpl(`${seed}${request.path}`, init);
    const body = await readBounded(response);
    return {
      statusCode: response.status,
      headers: safeResponseHeaders(response),
      body,
    };
  } finally {
    clearTimeout(timer);
  }
}

export async function proxyWithFailover(request, options = {}) {
  const seeds = options.seeds || [];
  const fetchImpl = options.fetchImpl || globalThis.fetch;
  const timeoutMs = Number.isFinite(options.timeoutMs) ? options.timeoutMs : DEFAULT_TIMEOUT_MS;
  if (!Array.isArray(seeds) || seeds.length !== 2) {
    throw new Error('exactly two seed endpoints are required');
  }
  if (typeof fetchImpl !== 'function') throw new Error('fetch implementation is required');

  let lastFailure;
  for (let i = 0; i < seeds.length; i++) {
    try {
      const result = await fetchOnce(seeds[i], request, { fetchImpl, timeoutMs });
      if (isRetryableStatus(result.statusCode) && i + 1 < seeds.length) {
        lastFailure = new UpstreamUnavailableError(`seed returned ${result.statusCode}`);
        continue;
      }
      return result;
    } catch (error) {
      lastFailure = error;
      if (i + 1 >= seeds.length) break;
    }
  }

  throw new UpstreamUnavailableError(lastFailure instanceof Error ? lastFailure.message : undefined);
}
