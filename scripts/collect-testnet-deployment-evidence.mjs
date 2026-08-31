import assert from 'node:assert/strict';

const DEFAULT_RPC_BASE_URL = 'https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com';

async function fetchJson(url) {
  const response = await fetch(url, { cache: 'no-store' });
  const bodyText = await response.text();
  let body = null;
  if (bodyText) {
    body = JSON.parse(bodyText);
  }
  return { status: response.status, body };
}

export async function collectPublicRpcSmoke({
  rpcBaseUrl = process.env.RPC_BASE_URL || DEFAULT_RPC_BASE_URL,
} = {}) {
  assert.equal(typeof rpcBaseUrl, 'string');
  assert.ok(rpcBaseUrl.startsWith('https://'), 'public RPC base URL must use HTTPS');

  const ready = await fetchJson(`${rpcBaseUrl}/ready`);
  const status = await fetchJson(`${rpcBaseUrl}/v1/status`);
  const explorerStatus = await fetchJson(`${rpcBaseUrl}/v1/explorer/status`);
  const faucetHealth = await fetchJson(`${rpcBaseUrl}/v1/faucet/health`);
  const visitors = await fetchJson(`${rpcBaseUrl}/v1/website/visitors`);

  if (ready.status !== 200) {
    throw new Error(`ready check failed: HTTP ${ready.status}`);
  }
  if (status.status !== 200 || status.body?.network !== 'sudharma') {
    throw new Error(`status check failed: HTTP ${status.status}`);
  }
  if (explorerStatus.status !== 200 || explorerStatus.body?.network !== 'sudharma') {
    throw new Error(`explorer status check failed: HTTP ${explorerStatus.status}`);
  }
  if (faucetHealth.status !== 200) {
    throw new Error(`faucet health check failed: HTTP ${faucetHealth.status}`);
  }
  if (visitors.status !== 200 || !Number.isSafeInteger(visitors.body?.total)) {
    throw new Error(`visitor counter check failed: HTTP ${visitors.status}`);
  }

  return {
    rpc_base_url: rpcBaseUrl,
    collected_at: new Date().toISOString(),
    ready: ready.body,
    status: status.body,
    explorer_status: explorerStatus.body,
    faucet_health: faucetHealth.body,
    visitor_total: visitors.body.total,
  };
}

if (process.argv[1]?.endsWith('collect-testnet-deployment-evidence.mjs')) {
  collectPublicRpcSmoke()
    .then((payload) => {
      process.stdout.write(`${JSON.stringify({ public_rpc_smoke: payload }, null, 2)}\n`);
    })
    .catch((error) => {
      process.stderr.write(`${error?.stack || error}\n`);
      process.exit(1);
    });
}
