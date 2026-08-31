"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { ExplorerCopyButton } from "@/components/explorer-copy-button";
import {
  ExplorerAPIError,
  ExplorerAddress,
  ExplorerBlock,
  ExplorerTransaction,
  directionLabel,
  fetchExplorerAddress,
  fetchExplorerBlock,
  fetchExplorerTransaction,
  formatExplorerTime,
  formatSUDH,
  shortHash,
  transactionDirection,
} from "@/lib/explorer-api";

type DetailProps = {
  apiBaseUrl: string;
};

type BlockDetailProps = DetailProps & {
  blockId: string;
};

type TransactionDetailProps = DetailProps & {
  transactionId: string;
};

type AddressDetailProps = DetailProps & {
  address: string;
};

type LoadState = "loading" | "ready" | "not-found" | "error" | "unavailable" | "missing";

function detailFailure(error: unknown, notFound: string, unavailable: string) {
  if (error instanceof ExplorerAPIError && error.status === 404) {
    return { state: "not-found" as LoadState, message: notFound };
  }
  const detail = error instanceof Error ? error.message : "Explorer API is unavailable";
  return { state: "error" as LoadState, message: `${unavailable} — ${detail}` };
}

function Loading({ children }: { children: string }) {
  return <p className="explorer-detail-state" aria-live="polite">{children}</p>;
}

function ErrorState({ message }: { message: string }) {
  return <p className="explorer-detail-state explorer-error" role="alert">{message}</p>;
}

function MissingConfig() {
  return <ErrorState message="Live explorer API is not configured." />;
}

function DetailHeader({ eyebrow, title, subtitle }: { eyebrow: string; title: string; subtitle?: string }) {
  return (
    <header className="explorer-detail-header">
      <p className="eyebrow">{eyebrow}</p>
      <h1>{title}</h1>
      {subtitle ? (
        <div className="explorer-detail-id-row">
          <p className="muted mono explorer-break">{subtitle}</p>
          <ExplorerCopyButton value={subtitle} label="Copy hash" />
        </div>
      ) : null}
      <Link className="text-link" href="/explorer">← Back to explorer</Link>
    </header>
  );
}

export function ExplorerBlockDetail({ apiBaseUrl, blockId }: BlockDetailProps) {
  const base = apiBaseUrl.trim();
  const id = blockId.trim();
  const [state, setState] = useState<LoadState>(base ? (id ? "loading" : "missing") : "unavailable");
  const [message, setMessage] = useState("");
  const [block, setBlock] = useState<ExplorerBlock | null>(null);

  useEffect(() => {
    if (!base) {
      setState("unavailable");
      return;
    }
    if (!id) {
      setState("missing");
      return;
    }
    let cancelled = false;
    setState("loading");
    void fetchExplorerBlock(base, id).then((payload) => {
      if (cancelled) return;
      setBlock(payload);
      setState("ready");
      setMessage("");
    }).catch((error: unknown) => {
      if (cancelled) return;
      const failure = detailFailure(error, "Block not found on the canonical chain.", "Block details unavailable");
      setState(failure.state);
      setMessage(failure.message);
    });
    return () => { cancelled = true; };
  }, [base, id]);

  if (state === "unavailable") return <MissingConfig />;
  if (state === "missing") return <ErrorState message="No block identifier was supplied." />;
  if (state === "loading") return <Loading>Loading block details…</Loading>;
  if (state === "not-found") return <p className="explorer-detail-state">{message}</p>;
  if (state === "error" || !block) return <ErrorState message={message || "Block details unavailable."} />;

  return (
    <article className="explorer-detail-card">
      <DetailHeader eyebrow="CANONICAL BLOCK" title={`Block #${block.height}`} subtitle={block.hash} />
      <dl className="explorer-detail-list">
        <div><dt>Timestamp</dt><dd>{formatExplorerTime(block.timestamp)}</dd></div>
        <div><dt>Previous hash</dt><dd><Link className="text-link mono explorer-break" href={`/explorer/block?id=${block.previous_hash}`}>{block.previous_hash || "—"}</Link></dd></div>
        <div><dt>Merkle root</dt><dd className="mono explorer-break">{block.merkle_root || "—"}</dd></div>
        <div><dt>Difficulty</dt><dd>{block.difficulty}</dd></div>
        <div><dt>Nonce</dt><dd>{block.nonce}</dd></div>
        <div><dt>Miner</dt><dd><Link className="text-link mono explorer-break" href={`/explorer/address?address=${block.miner_address}`}>{block.miner_address || "—"}</Link></dd></div>
        <div><dt>Transactions</dt><dd>{block.transaction_count}</dd></div>
      </dl>
      <section className="explorer-detail-section">
        <p className="eyebrow">TRANSACTIONS</p>
        {block.transactions?.length ? (
          <div className="explorer-detail-rows">
            {block.transactions.map((tx) => (
              <Link key={tx.id} className="explorer-detail-row explorer-row-rich" href={`/explorer/tx?id=${tx.id}`}>
                <strong className="mono explorer-break">{shortHash(tx.id)}</strong>
                <span>{formatSUDH(tx.amount)} · {shortHash(tx.from, 8)} → {shortHash(tx.to, 8)}</span>
              </Link>
            ))}
          </div>
        ) : <p className="muted">No transactions are recorded in this block.</p>}
      </section>
    </article>
  );
}

export function ExplorerTransactionDetail({ apiBaseUrl, transactionId }: TransactionDetailProps) {
  const base = apiBaseUrl.trim();
  const id = transactionId.trim();
  const [state, setState] = useState<LoadState>(base ? (id ? "loading" : "missing") : "unavailable");
  const [message, setMessage] = useState("");
  const [item, setItem] = useState<ExplorerTransaction | null>(null);

  useEffect(() => {
    if (!base) {
      setState("unavailable");
      return;
    }
    if (!id) {
      setState("missing");
      return;
    }
    let cancelled = false;
    setState("loading");
    void fetchExplorerTransaction(base, id).then((payload) => {
      if (cancelled) return;
      setItem(payload);
      setState("ready");
      setMessage("");
    }).catch((error: unknown) => {
      if (cancelled) return;
      const failure = detailFailure(error, "Transaction not found on the canonical chain or mempool.", "Transaction details unavailable");
      setState(failure.state);
      setMessage(failure.message);
    });
    return () => { cancelled = true; };
  }, [base, id]);

  if (state === "unavailable") return <MissingConfig />;
  if (state === "missing") return <ErrorState message="No transaction ID was supplied." />;
  if (state === "loading") return <Loading>Loading transaction details…</Loading>;
  if (state === "not-found") return <p className="explorer-detail-state">{message}</p>;
  if (state === "error" || !item) return <ErrorState message={message || "Transaction details unavailable."} />;

  const tx = item.transaction;
  const confirmed = item.status === "confirmed";
  return (
    <article className="explorer-detail-card">
      <DetailHeader eyebrow="TRANSACTION" title="Transaction detail" subtitle={tx.id} />
      <p className={`status-chip ${confirmed ? "status-completed" : "status-in-development"}`}>{confirmed ? "Confirmed" : "Pending"}</p>
      {!confirmed ? <p className="notice">This transaction is waiting in the seed-node mempool and is not yet included in a canonical block.</p> : null}
      <dl className="explorer-detail-list">
        <div><dt>Status</dt><dd>{confirmed ? "Success (confirmed)" : "Pending (mempool)"}</dd></div>
        <div><dt>From</dt><dd><Link className="text-link mono explorer-break" href={`/explorer/address?address=${tx.from}`}>{tx.from}</Link></dd></div>
        <div><dt>To</dt><dd><Link className="text-link mono explorer-break" href={`/explorer/address?address=${tx.to}`}>{tx.to}</Link></dd></div>
        <div><dt>Value</dt><dd>{formatSUDH(tx.amount)}</dd></div>
        <div><dt>Transaction fee</dt><dd>{formatSUDH(tx.fee)}</dd></div>
        <div><dt>Nonce</dt><dd>{tx.nonce}</dd></div>
        <div><dt>Confirmations</dt><dd>{item.confirmations}</dd></div>
        {confirmed && item.block_height !== undefined ? <div><dt>Block height</dt><dd><Link className="text-link" href={`/explorer/block?id=${item.block_hash || item.block_height}`}>{item.block_height}</Link></dd></div> : null}
        {confirmed && item.block_hash ? <div><dt>Block hash</dt><dd className="mono explorer-break">{item.block_hash}</dd></div> : null}
        {confirmed && item.block_timestamp ? <div><dt>Block time</dt><dd>{formatExplorerTime(item.block_timestamp)}</dd></div> : null}
      </dl>
      <div className="explorer-detail-actions">
        <ExplorerCopyButton value={tx.from} label="Copy sender" />
        <ExplorerCopyButton value={tx.to} label="Copy recipient" />
      </div>
    </article>
  );
}

export function ExplorerAddressDetail({ apiBaseUrl, address }: AddressDetailProps) {
  const base = apiBaseUrl.trim();
  const account = address.trim();
  const [state, setState] = useState<LoadState>(base ? (account ? "loading" : "missing") : "unavailable");
  const [message, setMessage] = useState("");
  const [data, setData] = useState<ExplorerAddress | null>(null);
  const [transactions, setTransactions] = useState<ExplorerTransaction[]>([]);
  const [cursor, setCursor] = useState("");
  const [paging, setPaging] = useState(false);
  const [pageNotice, setPageNotice] = useState("");

  useEffect(() => {
    if (!base) {
      setState("unavailable");
      return;
    }
    if (!account) {
      setState("missing");
      return;
    }
    let cancelled = false;
    setState("loading");
    void fetchExplorerAddress(base, account).then((payload) => {
      if (cancelled) return;
      setData(payload);
      setTransactions(payload.transactions ?? []);
      setCursor(payload.next_cursor ?? "");
      setState("ready");
      setMessage("");
    }).catch((error: unknown) => {
      if (cancelled) return;
      const failure = detailFailure(error, "Address data was not found.", "Address details unavailable");
      setState(failure.state);
      setMessage(failure.message);
    });
    return () => { cancelled = true; };
  }, [account, base]);

  async function loadOlder() {
    if (!cursor || paging) return;
    setPaging(true);
    setPageNotice("");
    try {
      const older = await fetchExplorerAddress(base, account, cursor);
      setData(older);
      setTransactions((current) => [...current, ...(older.transactions ?? [])]);
      setCursor(older.next_cursor ?? "");
    } catch (error) {
      if (error instanceof ExplorerAPIError && error.status === 409) {
        try {
          const fresh = await fetchExplorerAddress(base, account);
          setData(fresh);
          setTransactions(fresh.transactions ?? []);
          setCursor(fresh.next_cursor ?? "");
          setPageNotice("Chain changed while paging. History restarted from the canonical tip.");
        } catch (restartError) {
          setPageNotice(restartError instanceof Error ? `Address history restart failed — ${restartError.message}` : "Address history restart failed.");
        }
      } else {
        setPageNotice(error instanceof Error ? `Older transactions unavailable — ${error.message}` : "Older transactions unavailable.");
      }
    } finally {
      setPaging(false);
    }
  }

  if (state === "unavailable") return <MissingConfig />;
  if (state === "missing") return <ErrorState message="No address was supplied." />;
  if (state === "loading") return <Loading>Loading address details…</Loading>;
  if (state === "not-found") return <p className="explorer-detail-state">{message}</p>;
  if (state === "error" || !data) return <ErrorState message={message || "Address details unavailable."} />;

  return (
    <article className="explorer-detail-card">
      <DetailHeader eyebrow="ADDRESS" title="Account activity" subtitle={data.address} />
      <section className="explorer-metrics explorer-detail-metrics" aria-label="Address summary">
        <article><span>Balance</span><strong>{formatSUDH(data.balance)}</strong></article>
        <article><span>Confirmed nonce</span><strong>{data.confirmed_nonce}</strong></article>
        <article><span>Next nonce</span><strong>{data.next_nonce}</strong></article>
      </section>
      <div className="explorer-detail-actions">
        <ExplorerCopyButton value={data.address} label="Copy address" />
      </div>
      <section className="explorer-detail-section">
        <p className="eyebrow">CONFIRMED HISTORY</p>
        {transactions.length ? (
          <div className="explorer-detail-rows">
            {transactions.map((item) => {
              const direction = transactionDirection(item, data.address);
              const counterparty = direction === "sent" ? item.transaction.to : item.transaction.from;
              return (
                <Link key={`${item.transaction.id}-${item.block_height ?? "pending"}`} className="explorer-detail-row explorer-row-rich" href={`/explorer/tx?id=${item.transaction.id}`}>
                  <div>
                    <strong>{directionLabel(direction)}</strong>
                    <span className="mono explorer-break">{shortHash(item.transaction.id)}</span>
                  </div>
                  <div>
                    <span>{formatSUDH(item.transaction.amount)} · {direction === "sent" ? "to" : "from"} {shortHash(counterparty, 10)}</span>
                    <span>{item.confirmations} conf. · block {item.block_height ?? "pending"}</span>
                  </div>
                </Link>
              );
            })}
          </div>
        ) : <p className="muted">No confirmed transactions returned for this address.</p>}
        {cursor ? <button className="button secondary" type="button" disabled={paging} onClick={() => void loadOlder()}>{paging ? "Loading…" : "Load older transactions"}</button> : null}
        {pageNotice ? <p className="notice" aria-live="polite">{pageNotice}</p> : null}
      </section>
    </article>
  );
}
