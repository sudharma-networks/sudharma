export type FaucetInfo = {
  enabled: boolean;
  challenge_address: string;
  initial_grant_sudh: number;
  challenge_send_sudh: number;
  challenge_reward_sudh: number;
  max_rounds: number;
  cooldown_hours: number;
  testnet_only: boolean;
};

export type FaucetHealth = {
  ready: boolean;
  [key: string]: unknown;
};

export type FaucetInitialGrant = {
  address: string;
  amount_sudh: number;
  transaction_id: string;
  status: string;
};

export class FaucetAPIError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "FaucetAPIError";
    this.status = status;
  }
}

const ADDRESS_RE = /^[0-9a-f]{40}$/;

export function isValidSudharmaAddress(value: string) {
  return ADDRESS_RE.test(value);
}

function apiURL(base: string, path: string) {
  return `${base.replace(/\/$/, "")}${path}`;
}

async function readErrorMessage(response: Response) {
  try {
    const payload = await response.json() as { error?: string };
    return payload.error?.trim() || `Faucet API returned ${response.status}`;
  } catch {
    return `Faucet API returned ${response.status}`;
  }
}

async function readJSON<T>(base: string, path: string): Promise<T> {
  const response = await fetch(apiURL(base, path), { cache: "no-store" });
  if (!response.ok) {
    throw new FaucetAPIError(response.status, await readErrorMessage(response));
  }
  return response.json() as Promise<T>;
}

export function fetchFaucetInfo(base: string) {
  return readJSON<FaucetInfo>(base, "/v1/faucet/info");
}

export function fetchFaucetHealth(base: string) {
  return readJSON<FaucetHealth>(base, "/v1/faucet/health");
}

export async function requestFaucetInitialGrant(base: string, address: string): Promise<FaucetInitialGrant> {
  if (!isValidSudharmaAddress(address)) {
    throw new FaucetAPIError(400, "Enter a valid 40-character lowercase hex Sudharma address");
  }

  // text/plain avoids a CORS preflight while still sending JSON the Lambda accepts.
  const response = await fetch(apiURL(base, "/v1/faucet/request"), {
    method: "POST",
    headers: { "Content-Type": "text/plain;charset=UTF-8" },
    body: JSON.stringify({ address }),
    cache: "no-store",
  });

  if (!response.ok) {
    throw new FaucetAPIError(response.status, await readErrorMessage(response));
  }

  const payload = await response.json() as FaucetInitialGrant;
  if (
    typeof payload.address !== "string"
    || typeof payload.transaction_id !== "string"
    || typeof payload.amount_sudh !== "number"
    || typeof payload.status !== "string"
  ) {
    throw new FaucetAPIError(response.status, "invalid faucet response");
  }
  return payload;
}
