"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useState } from "react";
import {
  EXPLORER_DATA_SOURCES,
  ExplorerBlock,
  ExplorerSearchResult,
  ExplorerStatus,
  ExplorerTransaction,
  ExplorerAPIError,
  fetchExplorerBlocks,
  fetchExplorerMempool,
  fetchExplorerSearch,
  fetchExplorerStatus,
  fetchExplorerTransactions,
  formatExplorerTime,
  formatSUDH,
  shortHash,
} from "@/lib/explorer-api";

const MAX_SUPPLY_LABEL = "51,000,000,000 SUDH";

type ExplorerDashboardProps = {
  apiBaseUrl: string;
  pollIntervalMs?: number;
};

export function ExplorerDashboard({ apiBaseUrl, pollIntervalMs = 15_000 }: ExplorerDashboardProps) {
  const base = apiBaseUrl.trim();
  const [status, setStatus] = useState<ExplorerStatus | null>(null);
  const [blocks, setBlocks] = useState<ExplorerBlock[]>([]);
  const [transactions, setTransactions] = useState<ExplorerTransaction[]>([]);
  const [mempool, setMempool] = useState<ExplorerTransaction[]>([]);
  const [mempoolAvailable, setMempoolAvailable] = useState(true);
  const [connection, setConnection] = useState<"loading" | "connected" | "error" | "unconfigured">(base ? "loading" : "unconfigured");
  const [error, setError] = useState("");
  const [lastUpdated, setLastUpdated] = useState<number | null>(null);
  const [query, setQuery] = useState("");
  const [searching, setSearching] = useState(false);
  const [searchResult, setSearchResult] = useState<ExplorerSearchResult | null>(null);
  const [searchError, setSearchError] = useState("");

  const refresh = useCallback(async () => {
    if (!base) return;
    try {
      const [statusPayload, blocksPayload, transactionsPayload, mempoolResult] = await Promise.all([
        fetchExplorerStatus(base),
        fetchExplorerBlocks(base, 8),
        fetchExplorerTransactions(base, 8),
        fetchExplorerMempool(base, 8).then(
          (payload) => ({ ok: true as const, payload }),
          () => ({ ok: false as const, payload: { count: 0, transactions: [] } }),
        ),
      ]);
      setStatus(statusPayload);
      setBlocks(blocksPayload.blocks ?? []);
      setTransactions(transactionsPayload.transactions ?? []);
      if (mempoolResult.ok) {
        setMempool(mempoolResult.payload.transactions ?? []);
        setMempoolAvailable(true);
      } else {
        setMempool([]);
        setMempoolAvailable(false);
      }
      setConnection("connected");
      setError("");
      setLastUpdated(Date.now());
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
      setSearchResult(await fetchExplorerSearch(base, normalized));
    } catch (searchFailure) {
      if (searchFailure instanceof ExplorerAPIError && searchFailure.status === 404) {
        setSearchError("No canonical block, transaction, or address matched that search.");
      } else {
        setSearchError(searchFailure instanceof Error ? searchFailure.message : "Explorer search is unavailable.");
      }
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

  const dataSources = status?.data_sources?.length ? status.data_sources : [...EXPLORER_DATA_SOURCES];

  return (
    <div className="explorer-dashboard">
      <section className="explorer-search-panel">
        <div>
          <p className="eyebrow">LIVE TESTNET LOOKUP</p>
          <h2>Search the canonical chain.</h2>
          <p>Block height/hash · transaction ID · account address · pending mempool txs</p>
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
        {lastUpdated ? <span className="muted"> · Updated {new Date(lastUpdated).toLocaleTimeString()}</span> : null}
      </section>

      <section className="explorer-data-sources" aria-label="Data sources">
        <p className="eyebrow">DATA SOURCES</p>
        <p className="muted">Read-only explorer data is aggregated from testnet seed nodes with automatic failover, mempool state, and demand-miner wake signals.</p>
        <div className="explorer-source-tags">
          {dataSources.map((source) => (
            <span key={source} className="explorer-source-tag">{source}</span>
          ))}
        </div>
      </section>

      <section className="explorer-metrics" aria-label="Network summary">
        <article><span>Block height</span><strong>{status?.height ?? "—"}</strong></article>
        <article><span>Connected peers</span><strong>{status?.peers ?? "—"}</strong></article>
        <article><span>Mempool</span><strong>{status?.mempool ?? "—"}</strong></article>
        <article><span>Issued supply</span><strong>{status ? formatSUDH(status.issued_supply) : "—"}</strong></article>
        <article><span>Maximum supply</span><strong>{MAX_SUPPLY_LABEL}</strong></article>
        <article><span>Total work</span><strong>{status?.total_work ?? "—"}</strong></article>
      </section>

      <section className="explorer-grid explorer-grid-three">
        <div className="explorer-table-card">
          <div className="explorer-card-heading"><div><p className="eyebrow">BLOCKS</p><h2>Latest canonical blocks</h2></div><span className="muted">Auto-refreshing</span></div>
          {blocks.length ? <div className="explorer-rows">
            {blocks.map((block) => (
              <Link className="explorer-row explorer-row-rich" key={block.hash} href={`/explorer/block?id=${block.hash}`}>
                <div>
                  <strong>Block #{block.height}</strong>
                  <span className="mono">{shortHash(block.hash)}</span>
                </div>
                <div>
                  <span>{block.transaction_count} tx · miner {shortHash(block.miner_address, 8)}</span>
                  <span>{formatExplorerTime(block.timestamp)}</span>
                </div>
              </Link>
            ))}
          </div> : <p className="explorer-empty">No blocks returned by the public explorer API yet.</p>}
        </div>

        <div className="explorer-table-card">
          <div className="explorer-card-heading"><div><p className="eyebrow">TRANSACTIONS</p><h2>Latest confirmed activity</h2></div><span className="muted">Canonical only</span></div>
          {transactions.length ? <div className="explorer-rows">
            {transactions.map((item) => (
              <Link className="explorer-row explorer-row-rich" key={item.transaction.id} href={`/explorer/tx?id=${item.transaction.id}`}>
                <div>
                  <strong className="mono">{shortHash(item.transaction.id)}</strong>
                  <span>{formatSUDH(item.transaction.amount)} · fee {formatSUDH(item.transaction.fee)}</span>
                </div>
                <div>
                  <span className="mono">{shortHash(item.transaction.from, 8)} → {shortHash(item.transaction.to, 8)}</span>
                  <span>Block {item.block_height ?? "pending"} · {item.confirmations} conf.</span>
                </div>
              </Link>
            ))}
          </div> : <p className="explorer-empty">No confirmed transactions returned by the public explorer API yet.</p>}
        </div>

        <div className="explorer-table-card">
          <div className="explorer-card-heading"><div><p className="eyebrow">MEMPOOL</p><h2>Pending transactions</h2></div><span className="muted">{mempoolAvailable ? "Live" : "Count only"}</span></div>
          {mempool.length ? <div className="explorer-rows">
            {mempool.map((item) => (
              <Link className="explorer-row explorer-row-rich" key={item.transaction.id} href={`/explorer/tx?id=${item.transaction.id}`}>
                <div>
                  <strong className="mono">{shortHash(item.transaction.id)}</strong>
                  <span className="status-chip status-in-development">Pending</span>
                </div>
                <div>
                  <span>{formatSUDH(item.transaction.amount)} · fee {formatSUDH(item.transaction.fee)}</span>
                  <span className="mono">{shortHash(item.transaction.from, 8)} → {shortHash(item.transaction.to, 8)}</span>
                </div>
              </Link>
            ))}
          </div> : (
            <p className="explorer-empty">
              {status?.mempool ? `${status.mempool} pending transaction(s) reported by seed nodes.` : "Mempool is empty — no pending transactions."}
              {!mempoolAvailable ? " Detailed mempool listing will appear after the explorer mempool API is deployed on seeds." : ""}
            </p>
          )}
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
