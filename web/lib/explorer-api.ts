export const COIN_DECIMALS = 100_000_000;
export const EXPLORER_DATA_SOURCES = ["seed-1", "seed-2", "mempool", "demand-miner"] as const;

export type ExplorerStatus = {
  network: string;
  coin: string;
  symbol: string;
  height: number;
  tip_hash: string;
  total_work: string;
  peers: number;
  mempool: number;
  issued_supply: number;
  data_sources?: string[];
};

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

export type ExplorerMempool = {
  count: number;
  transactions: ExplorerTransaction[];
};

export type ExplorerSearchResult = {
  type: "block" | "transaction" | "address";
  path: string;
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

export function fetchExplorerStatus(base: string) {
  return readJSON<ExplorerStatus>(base, "/v1/explorer/status");
}

export function fetchExplorerBlocks(base: string, limit = 8) {
  return readJSON<{ blocks: ExplorerBlock[] }>(base, `/v1/explorer/blocks?limit=${limit}`);
}

export function fetchExplorerTransactions(base: string, limit = 8) {
  return readJSON<{ transactions: ExplorerTransaction[] }>(base, `/v1/explorer/transactions?limit=${limit}`);
}

export function fetchExplorerMempool(base: string, limit = 8) {
  return readJSON<ExplorerMempool>(base, `/v1/explorer/mempool?limit=${limit}`);
}

export function fetchExplorerSearch(base: string, query: string) {
  return readJSON<ExplorerSearchResult>(base, `/v1/explorer/search?q=${encodeURIComponent(query)}`);
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

export function shortHash(value: string, size = 12) {
  if (!value) return "—";
  if (value.length <= size) return value;
  return `${value.slice(0, size)}…`;
}

export function transactionDirection(
  item: ExplorerTransaction,
  focusAddress?: string,
): "sent" | "received" | "transfer" {
  if (!focusAddress) return "transfer";
  const { from, to } = item.transaction;
  if (from === focusAddress && to !== focusAddress) return "sent";
  if (to === focusAddress && from !== focusAddress) return "received";
  return "transfer";
}

export function directionLabel(direction: "sent" | "received" | "transfer") {
  if (direction === "sent") return "Sent";
  if (direction === "received") return "Received";
  return "Transfer";
}
