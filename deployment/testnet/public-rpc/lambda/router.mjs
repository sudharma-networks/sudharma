export const MAX_REQUEST_BYTES = 1024 * 1024;

export class RequestError extends Error {
  constructor(statusCode, message) {
    super(message);
    this.name = 'RequestError';
    this.statusCode = statusCode;
  }
}

const LOWER_HEX_40 = /^[0-9a-f]{40}$/;
const LOWER_HEX_64 = /^[0-9a-f]{64}$/;

export function validTransactionId(value) {
  return typeof value === 'string' && LOWER_HEX_64.test(value);
}

export function validAccountAddress(value) {
  return typeof value === 'string' && LOWER_HEX_40.test(value);
}

function reject(message = 'route not found', statusCode = 404) {
  throw new RequestError(statusCode, message);
}

export function matchRoute(methodInput, pathInput) {
  const method = String(methodInput || '').toUpperCase();
  const path = String(pathInput || '');

  if (!path.startsWith('/') || path.includes('\\') || /%2f|%5c|%2e/i.test(path)) {
    reject('invalid path', 400);
  }

  if (path === '/health') {
    if (method !== 'GET') reject('method not allowed', 405);
    return { kind: 'health', method, path };
  }
  if (path === '/ready') {
    if (method !== 'GET') reject('method not allowed', 405);
    return { kind: 'ready', method, path };
  }
  if (path === '/v1/status') {
    if (method !== 'GET') reject('method not allowed', 405);
    return { kind: 'status', method, path };
  }
  if (path === '/v1/transactions') {
    if (method !== 'POST') reject('method not allowed', 405);
    return { kind: 'submitTransaction', method, path };
  }
  if (path === '/v1/faucet/health') {
    if (method !== 'GET') reject('method not allowed', 405);
    return { kind: 'faucetHealth', method, path };
  }
  if (path === '/v1/faucet/info') {
    if (method !== 'GET') reject('method not allowed', 405);
    return { kind: 'faucetInfo', method, path };
  }
  if (path === '/v1/faucet/request') {
    if (method !== 'POST') reject('method not allowed', 405);
    return { kind: 'faucetInitial', method, path };
  }
  if (path === '/v1/faucet/challenge') {
    if (method !== 'POST') reject('method not allowed', 405);
    return { kind: 'faucetChallenge', method, path };
  }

  const accountPrefix = '/v1/accounts/';
  if (path.startsWith(accountPrefix)) {
    if (method !== 'GET') reject('method not allowed', 405);
    const address = path.slice(accountPrefix.length);
    if (!validAccountAddress(address)) reject('invalid account address', 400);
    return { kind: 'account', method, path, address };
  }

  const transactionPrefix = '/v1/transactions/';
  if (path.startsWith(transactionPrefix)) {
    if (method !== 'GET') reject('method not allowed', 405);
    const transactionId = path.slice(transactionPrefix.length);
    if (!validTransactionId(transactionId)) reject('invalid transaction id', 400);
    return { kind: 'transactionStatus', method, path, transactionId };
  }

  reject();
}

function bodyBuffer(event) {
  if (event?.body == null || event.body === '') return Buffer.alloc(0);
  if (typeof event.body !== 'string') throw new RequestError(400, 'invalid request body');
  let body;
  try {
    body = Buffer.from(event.body, event.isBase64Encoded ? 'base64' : 'utf8');
  } catch {
    throw new RequestError(400, 'invalid request body encoding');
  }
  if (body.length > MAX_REQUEST_BYTES) throw new RequestError(413, 'request body too large');
  return body;
}

export function normalizeEvent(event) {
  const method = event?.requestContext?.http?.method;
  const path = event?.rawPath;
  if (typeof method !== 'string' || typeof path !== 'string') {
    throw new RequestError(400, 'invalid API Gateway event');
  }

  const route = matchRoute(method, path);
  const body = bodyBuffer(event);
  const bodyAllowed = route.kind === 'submitTransaction' || route.kind === 'faucetInitial' || route.kind === 'faucetChallenge';
  if (!bodyAllowed && body.length !== 0) {
    throw new RequestError(400, 'request body not allowed');
  }
  if ((route.kind === 'faucetInitial' || route.kind === 'faucetChallenge') && body.length === 0) {
    throw new RequestError(400, 'request body is required');
  }

  const headers = {};
  const inputHeaders = event.headers && typeof event.headers === 'object' ? event.headers : {};
  const contentType = Object.entries(inputHeaders).find(([name]) => name.toLowerCase() === 'content-type')?.[1];
  if (typeof contentType === 'string' && contentType.length <= 200) {
    headers['content-type'] = contentType;
  }

  return {
    ...route,
    body,
    headers,
  };
}
