const DEFAULT_RPC_BASE_URL = 'https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com';

async function fetchJson(url) {
  const response = await fetch(url, { cache: 'no-store' });
  const bodyText = await response.text();
  let body;
  try {
    body = bodyText ? JSON.parse(bodyText) : null;
  } catch {
    throw new Error(`invalid JSON from ${url}`);
  }
  return { status: response.status, body };
}

export async function runExplorerLiveCheck({
  rpcBaseUrl = process.env.RPC_BASE_URL || DEFAULT_RPC_BASE_URL,
} = {}) {
  const status = await fetchJson(`${rpcBaseUrl}/v1/explorer/status`);
  if (status.status !== 200 || status.body?.network !== 'sudharma') {
    throw new Error(`explorer status check failed: HTTP ${status.status}`);
  }
  if (!Number.isInteger(status.body?.height) || status.body.height < 1) {
    throw new Error('explorer status returned invalid height');
  }

  const blocks = await fetchJson(`${rpcBaseUrl}/v1/explorer/blocks?limit=3`);
  if (blocks.status !== 200 || !Array.isArray(blocks.body?.blocks) || blocks.body.blocks.length < 1) {
    throw new Error('explorer blocks feed is unavailable');
  }

  const transactions = await fetchJson(`${rpcBaseUrl}/v1/explorer/transactions?limit=3`);
  if (transactions.status !== 200 || !Array.isArray(transactions.body?.transactions)) {
    throw new Error('explorer transactions feed is unavailable');
  }

  const mempool = await fetchJson(`${rpcBaseUrl}/v1/explorer/mempool?limit=8`);
  if (mempool.status !== 200 || !Number.isInteger(mempool.body?.count) || !Array.isArray(mempool.body?.transactions)) {
    throw new Error('explorer mempool feed is unavailable');
  }

  const tipHash = String(status.body.tip_hash || '');
  const search = await fetchJson(`${rpcBaseUrl}/v1/explorer/search?q=${encodeURIComponent(tipHash)}`);
  if (search.status !== 200 || search.body?.type !== 'block') {
    throw new Error('explorer search did not resolve the current tip hash');
  }

  return {
    ok: true,
    height: status.body.height,
    tip_hash: tipHash,
    block_count: blocks.body.blocks.length,
    transaction_count: transactions.body.transactions.length,
    mempool_count: mempool.body.count,
    search_path: search.body.path,
    api_base_url: rpcBaseUrl,
  };
}

if (process.argv[1]?.endsWith('explorer-live-check.mjs')) {
  runExplorerLiveCheck()
    .then((result) => {
      process.stdout.write(`${JSON.stringify(result)}\n`);
    })
    .catch((error) => {
      process.stderr.write(`${error?.stack || error}\n`);
      process.exit(1);
    });
}
