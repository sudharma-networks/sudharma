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
const DECIMAL_UINT64 = /^[0-9]{1,20}$/;
const EXPLORER_CURSOR = /^[A-Za-z0-9_-]{1,256}$/;
const UINT64_MAX = 18446744073709551615n;
const MAX_EXPLORER_QUERY_BYTES = 1024;

export function validTransactionId(value) {
  return typeof value === 'string' && LOWER_HEX_64.test(value);
}

export function validAccountAddress(value) {
  return typeof value === 'string' && LOWER_HEX_40.test(value);
}

function validUint64(value) {
  if (typeof value !== 'string' || !DECIMAL_UINT64.test(value)) return false;
  try {
    return BigInt(value) <= UINT64_MAX;
  } catch {
    return false;
  }
}

function validBlockIdentifier(value) {
  return validUint64(value) || (typeof value === 'string' && LOWER_HEX_64.test(value));
}

function validExplorerSearch(value) {
  return validUint64(value)
    || (typeof value === 'string' && LOWER_HEX_40.test(value))
    || (typeof value === 'string' && LOWER_HEX_64.test(value));
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
  if (path === '/v1/mempool') {
    if (method !== 'GET') reject('method not allowed', 405);
    return { kind: 'mempool', method, path };
  }
  if (path === '/v1/transactions') {
    if (method !== 'POST') reject('method not allowed', 405);
    return { kind: 'submitTransaction', method, path };
  }
  if (path === '/v1/miner/wake') {
    if (method !== 'POST') reject('method not allowed', 405);
    return { kind: 'minerWake', method, path };
  }
  if (path === '/v1/faucet/info') {
    if (method !== 'GET') reject('method not allowed', 405);
    return { kind: 'faucetInfo', method, path };
  }
  if (path === '/v1/faucet/health') {
    if (method !== 'GET') reject('method not allowed', 405);
    return { kind: 'faucetHealth', method, path };
  }
  if (path === '/v1/faucet/diagnostics') {
    if (method !== 'GET') reject('method not allowed', 405);
    return { kind: 'faucetDiagnostics', method, path };
  }
  if (path === '/v1/faucet/request') {
    if (method !== 'POST') reject('method not allowed', 405);
    return { kind: 'faucetInitial', method, path };
  }
  if (path === '/v1/faucet/challenge') {
    if (method !== 'POST') reject('method not allowed', 405);
    return { kind: 'faucetChallenge', method, path };
  }
  if (path === '/v1/website/visitors') {
    if (method === 'GET') return { kind: 'websiteVisitorsRead', method, path };
    if (method === 'POST') return { kind: 'websiteVisitorsRecord', method, path };
    reject('method not allowed', 405);
  }

  if (path === '/v1/explorer/status') {
    if (method !== 'GET') reject('method not allowed', 405);
    return { kind: 'explorerStatus', method, path };
  }
  if (path === '/v1/explorer/blocks') {
    if (method !== 'GET') reject('method not allowed', 405);
    return { kind: 'explorerBlocks', method, path };
  }
  if (path === '/v1/explorer/transactions') {
    if (method !== 'GET') reject('method not allowed', 405);
    return { kind: 'explorerTransactions', method, path };
  }
  if (path === '/v1/explorer/search') {
    if (method !== 'GET') reject('method not allowed', 405);
    return { kind: 'explorerSearch', method, path };
  }

  const explorerBlockPrefix = '/v1/explorer/blocks/';
  if (path.startsWith(explorerBlockPrefix)) {
    if (method !== 'GET') reject('method not allowed', 405);
    const blockId = path.slice(explorerBlockPrefix.length);
    if (!validBlockIdentifier(blockId)) reject('invalid block identifier', 400);
    return { kind: 'explorerBlock', method, path, blockId };
  }

  const explorerTransactionPrefix = '/v1/explorer/transactions/';
  if (path.startsWith(explorerTransactionPrefix)) {
    if (method !== 'GET') reject('method not allowed', 405);
    const transactionId = path.slice(explorerTransactionPrefix.length);
    if (!validTransactionId(transactionId)) reject('invalid transaction id', 400);
    return { kind: 'explorerTransaction', method, path, transactionId };
  }

  const explorerAddressPrefix = '/v1/explorer/addresses/';
  if (path.startsWith(explorerAddressPrefix)) {
    if (method !== 'GET') reject('method not allowed', 405);
    const address = path.slice(explorerAddressPrefix.length);
    if (!validAccountAddress(address)) reject('invalid account address', 400);
    return { kind: 'explorerAddress', method, path, address };
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

function explorerQueryRules(kind) {
  switch (kind) {
    case 'explorerBlocks':
      return new Map([
        ['limit', (value) => /^[0-9]{1,3}$/.test(value) && Number(value) >= 1 && Number(value) <= 100],
        ['before', validUint64],
      ]);
    case 'explorerTransactions':
    case 'explorerAddress':
      return new Map([
        ['limit', (value) => /^[0-9]{1,3}$/.test(value) && Number(value) >= 1 && Number(value) <= 100],
        ['before_height', validUint64],
        ['cursor', (value) => EXPLORER_CURSOR.test(value)],
      ]);
    case 'explorerSearch':
      return new Map([['q', validExplorerSearch]]);
    case 'explorerStatus':
    case 'explorerBlock':
    case 'explorerTransaction':
      return new Map();
    default:
      return null;
  }
}

function normalizeExplorerQuery(route, rawInput) {
  const rules = explorerQueryRules(route.kind);
  if (rules === null) return '';
  if (rawInput == null || rawInput === '') {
    if (route.kind === 'explorerSearch') reject('search query is required', 400);
    return '';
  }
  if (typeof rawInput !== 'string' || Buffer.byteLength(rawInput, 'utf8') > MAX_EXPLORER_QUERY_BYTES) {
    reject('invalid explorer query', 400);
  }

  const params = new URLSearchParams(rawInput);
  const seen = new Set();
  for (const [key, value] of params.entries()) {
    const validator = rules.get(key);
    if (!validator) reject('unsupported explorer query parameter', 400);
    if (seen.has(key)) reject('duplicate explorer query parameter', 400);
    seen.add(key);
    if (!validator(value)) reject('invalid explorer query parameter', 400);
  }

  if (route.kind === 'explorerSearch' && !seen.has('q')) {
    reject('search query is required', 400);
  }
  if ((route.kind === 'explorerTransactions' || route.kind === 'explorerAddress')
      && seen.has('cursor') && seen.has('before_height')) {
    reject('cursor and before_height cannot be combined', 400);
  }

  return params.toString();
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
  const queryString = normalizeExplorerQuery(route, event?.rawQueryString);
  const body = bodyBuffer(event);
  const bodyAllowed = route.kind === 'submitTransaction'
    || route.kind === 'faucetInitial'
    || route.kind === 'faucetChallenge'
    || route.kind === 'websiteVisitorsRecord';
  if (!bodyAllowed && body.length !== 0) {
    throw new RequestError(400, 'request body not allowed');
  }
  if ((route.kind === 'faucetInitial' || route.kind === 'faucetChallenge' || route.kind === 'websiteVisitorsRecord') && body.length === 0) {
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
    queryString,
    body,
    headers,
  };
}
