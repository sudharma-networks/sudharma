"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useState } from "react";

const COIN_DECIMALS = 100_000_000;
const MAX_SUPPLY_LABEL = "51,000,000,000 SUDH";

type ExplorerStatus = {
  network: string;
  coin: string;
  symbol: string;
  height: number;
  tip_hash: string;
  total_work: string;
  peers: number;
  mempool: number;
  issued_supply: number;
};

type ExplorerBlock = {
  height: number;
  hash: string;
  timestamp: number;
  previous_hash: string;
  merkle_root: string;
  difficulty: number;
  nonce: number;
  miner_address: string;
  transaction_count: number;
};

type ExplorerTransaction = {
  transaction: {
    id: string;
    from: string;
    to: string;
    amount: number;
    fee: number;
    nonce: number;
  };
  status: string;
  block_height?: number;
  block_hash?: string;
  block_timestamp?: number;
  confirmations: number;
};

type SearchResult = {
  type: "block" | "transaction" | "address";
  path: string;
};

type ExplorerDashboardProps = {
  apiBaseUrl: string;
  pollIntervalMs?: number;
};

function apiURL(base: string, path: string) {
  return `${base.replace(/\/$/, "")}${path}`;
}

async function readJSON<T>(base: string, path: string): Promise<T> {
  const response = await fetch(apiURL(base, path), { cache: "no-store" });
  if (!response.ok) {
    throw new Error(`Explorer API returned ${response.status}`);
  }
  return response.json() as Promise<T>;
}

function shortHash(value: string, size = 12) {
  if (!value) return "—";
  if (value.length <= size) return value;
  return `${value.slice(0, size)}…`;
}

function formatSUDH(baseUnits: number) {
  if (!Number.isFinite(baseUnits)) return "—";
  return `${(baseUnits / COIN_DECIMALS).toLocaleString(undefined, { maximumFractionDigits: 8 })} SUDH`;
}

function formatTime(unixSeconds?: number) {
  if (!unixSeconds) return "—";
  return new Date(unixSeconds * 1000).toLocaleString();
}

export function ExplorerDashboard({ apiBaseUrl, pollIntervalMs = 15_000 }: ExplorerDashboardProps) {
  const base = apiBaseUrl.trim();
  const [status, setStatus] = useState<ExplorerStatus | null>(null);
  const [blocks, setBlocks] = useState<ExplorerBlock[]>([]);
  const [transactions, setTransactions] = useState<ExplorerTransaction[]>([]);
  const [connection, setConnection] = useState<"loading" | "connected" | "error" | "unconfigured">(base ? "loading" : "unconfigured");
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [searching, setSearching] = useState(false);
  const [searchResult, setSearchResult] = useState<SearchResult | null>(null);
  const [searchError, setSearchError] = useState("");

  const refresh = useCallback(async () => {
    if (!base) return;
    try {
      const [statusPayload, blocksPayload, transactionsPayload] = await Promise.all([
        readJSON<ExplorerStatus>(base, "/v1/explorer/status"),
        readJSON<{ blocks: ExplorerBlock[] }>(base, "/v1/explorer/blocks?limit=8"),
        readJSON<{ transactions: ExplorerTransaction[] }>(base, "/v1/explorer/transactions?limit=8")
      ]);
      setStatus(statusPayload);
      setBlocks(blocksPayload.blocks ?? []);
      setTransactions(transactionsPayload.transactions ?? []);
      setConnection("connected");
      setError("");
    } catch (refreshError) {
      setConnection("error");
      setError(refreshError instanceof Error ? refreshError.message : "Explorer API is unavailable");
    }
  }, [base]);

  useEffect(() => {
    if (!base) {
      setConnection("unconfigured");
      return;
    }
    void refresh();
    const timer = window.setInterval(() => void refresh(), pollIntervalMs);
    return () => window.clearInterval(timer);
  }, [base, pollIntervalMs, refresh]);

  async function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalized = query.trim();
    setSearchResult(null);
    setSearchError("");
    if (!base) {
      setSearchError("Live explorer API is not configured.");
      return;
    }
    if (!normalized) {
      setSearchError("Enter a block height, block hash, transaction ID, or address.");
      return;
    }
    setSearching(true);
    try {
      const response = await fetch(apiURL(base, `/v1/explorer/search?q=${encodeURIComponent(normalized)}`), { cache: "no-store" });
      if (response.status === 404) {
        setSearchError("No canonical block, transaction, or address matched that search.");
        return;
      }
      if (!response.ok) {
        throw new Error(`Explorer search returned ${response.status}`);
      }
      setSearchResult(await response.json() as SearchResult);
    } catch (searchFailure) {
      setSearchError(searchFailure instanceof Error ? searchFailure.message : "Explorer search is unavailable.");
    } finally {
      setSearching(false);
    }
  }

  if (!base) {
    return (
      <section className="explorer-unavailable" aria-live="polite">
        <strong>Live explorer API is not configured.</strong>
        <p>The explorer frontend is ready, but it will not invent chain height, transactions, supply, or peer data. Public values will appear only after the reviewed read-only testnet API is deployed.</p>
        <p className="mono">Maximum supply (hard cap): {MAX_SUPPLY_LABEL}</p>
      </section>
    );
  }

  return (
    <div className="explorer-dashboard">
      <section className="explorer-search-panel">
        <div>
          <p className="eyebrow">LIVE TESTNET LOOKUP</p>
          <h2>Search the canonical chain.</h2>
          <p>Block height/hash · transaction ID · account address</p>
        </div>
        <form className="explorer-search" onSubmit={submitSearch}>
          <label htmlFor="explorer-search-input">Search blockchain</label>
          <div>
            <input
              id="explorer-search-input"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Height, block hash, transaction ID, or address"
              autoComplete="off"
              spellCheck={false}
            />
            <button className="button" type="submit" disabled={searching}>{searching ? "Searching…" : "Search"}</button>
          </div>
        </form>
        {searchResult ? <p className="search-result"><Link className="text-link" href={searchResult.path}>Open {searchResult.type} result →</Link></p> : null}
        {searchError ? <p className="explorer-error" role="alert">{searchError}</p> : null}
      </section>

      <section className="explorer-connection" aria-live="polite">
        <span className={`status-dot ${connection === "error" ? "status-dot-error" : ""}`} />
        {connection === "connected" ? "Connected to live testnet data" : connection === "loading" ? "Connecting to live testnet data…" : `Live testnet data unavailable${error ? ` — ${error}` : ""}`}
      </section>

      <section className="explorer-metrics" aria-label="Network summary">
        <article><span>Block height</span><strong>{status?.height ?? "—"}</strong></article>
        <article><span>Connected peers</span><strong>{status?.peers ?? "—"}</strong></article>
        <article><span>Mempool</span><strong>{status?.mempool ?? "—"}</strong></article>
        <article><span>Issued supply</span><strong>{status ? formatSUDH(status.issued_supply) : "—"}</strong></article>
        <article><span>Maximum supply</span><strong>{MAX_SUPPLY_LABEL}</strong></article>
        <article><span>Total work</span><strong>{status?.total_work ?? "—"}</strong></article>
      </section>

      <section className="explorer-grid">
        <div className="explorer-table-card">
          <div className="explorer-card-heading"><div><p className="eyebrow">BLOCKS</p><h2>Latest canonical blocks</h2></div><span className="muted">Auto-refreshing</span></div>
          {blocks.length ? <div className="explorer-rows">
            {blocks.map((block) => (
              <Link className="explorer-row" key={block.hash} href={`/explorer/block?id=${block.hash}`}>
                <div><strong>Block #{block.height}</strong><span className="mono">{shortHash(block.hash)}</span></div>
                <div><span>{block.transaction_count} tx</span><span>{formatTime(block.timestamp)}</span></div>
              </Link>
            ))}
          </div> : <p className="explorer-empty">No blocks returned by the public explorer API yet.</p>}
        </div>

        <div className="explorer-table-card">
          <div className="explorer-card-heading"><div><p className="eyebrow">TRANSACTIONS</p><h2>Latest confirmed activity</h2></div><span className="muted">Canonical only</span></div>
          {transactions.length ? <div className="explorer-rows">
            {transactions.map((item) => (
              <Link className="explorer-row" key={item.transaction.id} aria-label={shortHash(item.transaction.id)} href={`/explorer/tx?id=${item.transaction.id}`}>
                <div><strong className="mono">{shortHash(item.transaction.id)}</strong><span>{formatSUDH(item.transaction.amount)}</span></div>
                <div><span>Block {item.block_height ?? "pending"}</span><span>{item.confirmations} conf.</span></div>
              </Link>
            ))}
          </div> : <p className="explorer-empty">No confirmed transactions returned by the public explorer API yet.</p>}
        </div>
      </section>

      <section className="explorer-tip">
        <span>Canonical tip</span>
        <strong className="mono">{status ? shortHash(status.tip_hash, 24) : "—"}</strong>
        <span>{status?.symbol ?? "SUDH"} · pre-mainnet public testnet</span>
      </section>
    </div>
  );
}
