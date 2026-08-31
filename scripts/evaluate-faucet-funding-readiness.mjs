const COIN = 100_000_000;
const INITIAL_GRANT_SUDH = 100;

export function requiredInitialGrantBalance() {
  const amount = INITIAL_GRANT_SUDH * COIN;
  const fee = Math.floor((amount * 10) / 10_000);
  return amount + fee;
}

export function evaluateFaucetFundingReadiness({
  faucetInfo,
  faucetDiagnostics,
  signerAccount,
} = {}) {
  const enabled = faucetInfo?.enabled === true;
  const balance = Number(signerAccount?.balance);
  const required = requiredInitialGrantBalance();
  const ready = faucetDiagnostics?.ready === true;
  const shortfall = Number.isFinite(balance) ? Math.max(0, required - balance) : required;

  let reason = 'ready';
  if (!enabled) reason = 'faucet_disabled';
  else if (!Number.isFinite(balance)) reason = 'signer_balance_unavailable';
  else if (balance < required) reason = 'faucet_needs_funding';
  else if (!ready) reason = 'faucet_not_ready';

  return {
    enabled,
    ready: enabled && balance >= required && ready,
    signer_balance: Number.isFinite(balance) ? balance : null,
    required_balance: required,
    shortfall,
    reason,
    should_refill: enabled && Number.isFinite(balance) && balance < required,
    blocks_to_refill_estimate: shortfall > 0 ? Math.ceil(shortfall / (50 * COIN)) : 0,
  };
}

if (process.argv[1]?.endsWith('evaluate-faucet-funding-readiness.mjs')) {
  const rpcBaseUrl = process.env.RPC_BASE_URL || 'https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com';
  const signer = process.env.FAUCET_SIGNER || '9ccdc094489874bed888ffe4bdf9b8298f4c5131';

  async function fetchJson(path) {
    const response = await fetch(`${rpcBaseUrl}${path}`, { cache: 'no-store' });
    const body = await response.json().catch(() => ({}));
    return { status: response.status, body };
  }

  Promise.all([
    fetchJson('/v1/faucet/info'),
    fetchJson('/v1/faucet/diagnostics'),
    fetchJson(`/v1/accounts/${signer}`),
  ]).then(([info, diagnostics, account]) => {
    const result = evaluateFaucetFundingReadiness({
      faucetInfo: info.body,
      faucetDiagnostics: diagnostics.body,
      signerAccount: account.body,
    });
    process.stdout.write(`${JSON.stringify({ funding: result }, null, 2)}\n`);
    process.exit(result.ready ? 0 : 2);
  }).catch((error) => {
    process.stderr.write(`${error?.stack || error}\n`);
    process.exit(1);
  });
}
