export const COIN_DECIMALS = 100_000_000;

export type ExplorerTransactionView = {
  id: string;
  from: string;
  to: string;
  amount: number;
  fee: number;
  nonce: number;
};

export type ExplorerTransaction = {
  transaction: ExplorerTransactionView;
  status: "pending" | "confirmed" | string;
  block_height?: number;
  block_hash?: string;
  block_timestamp?: number;
  confirmations: number;
};

export type ExplorerBlock = {
  height: number;
  hash: string;
  timestamp: number;
  previous_hash: string;
  merkle_root: string;
  difficulty: number;
  nonce: number;
  miner_address: string;
  transaction_count: number;
  transactions?: ExplorerTransactionView[];
};

export type ExplorerAddress = {
  address: string;
  balance: number;
  confirmed_nonce: number;
  next_nonce: number;
  transactions: ExplorerTransaction[];
  next_cursor?: string;
};

export class ExplorerAPIError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ExplorerAPIError";
    this.status = status;
  }
}

function apiURL(base: string, path: string) {
  return `${base.replace(/\/$/, "")}${path}`;
}

async function readJSON<T>(base: string, path: string): Promise<T> {
  const response = await fetch(apiURL(base, path), { cache: "no-store" });
  if (!response.ok) {
    let detail = "";
    try {
      const payload = await response.json() as { error?: string };
      detail = payload.error?.trim() ?? "";
    } catch {
      detail = "";
    }
    throw new ExplorerAPIError(response.status, detail || `Explorer API returned ${response.status}`);
  }
  return response.json() as Promise<T>;
}

export function fetchExplorerBlock(base: string, blockId: string) {
  return readJSON<ExplorerBlock>(base, `/v1/explorer/blocks/${encodeURIComponent(blockId)}`);
}

export function fetchExplorerTransaction(base: string, transactionId: string) {
  return readJSON<ExplorerTransaction>(base, `/v1/explorer/transactions/${encodeURIComponent(transactionId)}`);
}

export function fetchExplorerAddress(base: string, address: string, cursor = "") {
  const cursorQuery = cursor ? `&cursor=${encodeURIComponent(cursor)}` : "";
  return readJSON<ExplorerAddress>(base, `/v1/explorer/addresses/${encodeURIComponent(address)}?limit=20${cursorQuery}`);
}

export function formatSUDH(baseUnits: number) {
  if (!Number.isFinite(baseUnits)) return "—";
  return `${(baseUnits / COIN_DECIMALS).toLocaleString(undefined, { maximumFractionDigits: 8 })} SUDH`;
}

export function formatExplorerTime(unixSeconds?: number) {
  if (!unixSeconds) return "—";
  return new Date(unixSeconds * 1000).toLocaleString();
}
