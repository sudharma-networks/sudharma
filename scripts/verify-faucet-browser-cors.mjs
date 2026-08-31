#!/usr/bin/env node
/**
 * Read-only check that the live public faucet answers browser CORS.
 * Exit 0 when Access-Control-Allow-Origin is present on GET /v1/faucet/info
 * and OPTIONS /v1/faucet/request returns 2xx.
 */
const base = (process.argv[2] || 'https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com').replace(/\/$/, '');

function fail(message) {
  process.stderr.write(`${message}\n`);
  process.exit(1);
}

const info = await fetch(`${base}/v1/faucet/info`, {
  headers: { Origin: 'https://example.test' },
});
if (!info.ok) fail(`GET /v1/faucet/info returned ${info.status}`);
const infoCors = info.headers.get('access-control-allow-origin');
if (infoCors !== '*') {
  fail(`GET /v1/faucet/info missing browser CORS (got ${JSON.stringify(infoCors)}). Redeploy Testnet Public RPC from the Stage 6 branch.`);
}

const options = await fetch(`${base}/v1/faucet/request`, {
  method: 'OPTIONS',
  headers: {
    Origin: 'https://example.test',
    'Access-Control-Request-Method': 'POST',
    'Access-Control-Request-Headers': 'content-type',
  },
});
if (options.status < 200 || options.status >= 300) {
  fail(`OPTIONS /v1/faucet/request returned ${options.status}. Redeploy Testnet Public RPC from the Stage 6 branch.`);
}
const optionsCors = options.headers.get('access-control-allow-origin');
if (optionsCors !== '*') {
  fail(`OPTIONS /v1/faucet/request missing Access-Control-Allow-Origin (got ${JSON.stringify(optionsCors)}).`);
}

process.stdout.write(`PASS faucet browser CORS at ${base}\n`);
