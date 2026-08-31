"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useState } from "react";
import {
  FaucetAPIError,
  FaucetInfo,
  FaucetInitialGrant,
  fetchFaucetHealth,
  fetchFaucetInfo,
  isValidSudharmaAddress,
  requestFaucetInitialGrant,
} from "@/lib/faucet-api";

type FaucetPanelProps = {
  apiBaseUrl: string;
};

export function FaucetPanel({ apiBaseUrl }: FaucetPanelProps) {
  const base = apiBaseUrl.trim();
  const [info, setInfo] = useState<FaucetInfo | null>(null);
  const [ready, setReady] = useState(false);
  const [loadState, setLoadState] = useState<"loading" | "ready" | "error" | "unconfigured">(base ? "loading" : "unconfigured");
  const [loadError, setLoadError] = useState("");
  const [address, setAddress] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState("");
  const [grant, setGrant] = useState<FaucetInitialGrant | null>(null);

  const refresh = useCallback(async () => {
    if (!base) return;
    try {
      const [infoPayload, healthPayload] = await Promise.all([
        fetchFaucetInfo(base),
        fetchFaucetHealth(base).catch(() => ({ ready: false })),
      ]);
      setInfo(infoPayload);
      setReady(Boolean(healthPayload.ready));
      setLoadState("ready");
      setLoadError("");
    } catch (error) {
      setInfo(null);
      setReady(false);
      setLoadState("error");
      setLoadError(error instanceof FaucetAPIError ? error.message : "Unable to reach the public faucet");
    }
  }, [base]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError("");
    setGrant(null);

    const normalized = address.trim().toLowerCase();
    if (!isValidSudharmaAddress(normalized)) {
      setFormError("Enter a valid 40-character lowercase hex Sudharma address");
      return;
    }
    if (!info?.enabled || !ready) {
      setFormError("The faucet is currently unavailable. Try again when status shows ready.");
      return;
    }

    setSubmitting(true);
    try {
      const result = await requestFaucetInitialGrant(base, normalized);
      setGrant(result);
      setAddress(normalized);
    } catch (error) {
      setFormError(error instanceof FaucetAPIError ? error.message : "Unable to request test SUDH");
    } finally {
      setSubmitting(false);
    }
  }

  if (loadState === "unconfigured") {
    return <p className="notice">Faucet API is not configured for this build.</p>;
  }

  if (loadState === "loading") {
    return <p className="muted" aria-live="polite">Checking faucet status…</p>;
  }

  if (loadState === "error" || !info) {
    return (
      <section className="notice danger" aria-live="polite">
        <strong>Faucet unavailable.</strong>
        <p>{loadError || "The public faucet could not be reached."}</p>
        <button type="button" className="button secondary small" onClick={() => void refresh()}>Retry</button>
      </section>
    );
  }

  const canRequest = info.enabled && ready && !submitting;

  return (
    <div className="faucet-panel">
      <section className="notice">
        <strong>Testnet only.</strong>
        <p>Test SUDH has no mainnet value. Never paste a seed phrase or private key here. This page requests an initial grant only; challenge rounds remain in the Android wallet.</p>
      </section>

      <section className="faucet-status-card" aria-live="polite">
        <div className="explorer-connection">
          <span className={`status-dot${info.enabled && ready ? "" : " status-dot-error"}`} aria-hidden="true" />
          <span>
            {info.enabled && ready
              ? "Faucet enabled and ready"
              : info.enabled
                ? "Faucet enabled but not ready"
                : "Faucet temporarily disabled"}
          </span>
        </div>
        <div className="explorer-metrics">
          <article><span>Initial grant</span><strong>{info.initial_grant_sudh} SUDH</strong></article>
          <article><span>Challenge send</span><strong>{info.challenge_send_sudh} SUDH</strong></article>
          <article><span>Challenge reward</span><strong>{info.challenge_reward_sudh} SUDH</strong></article>
        </div>
      </section>

      <section className="faucet-form-card">
        <h2>Request initial test SUDH</h2>
        <p className="muted">One initial grant per address. Enter the receiving wallet address from the Sudharma Android wallet or another compatible client.</p>
        <form className="faucet-form" onSubmit={onSubmit}>
          <label htmlFor="faucet-address">Sudharma address</label>
          <div>
            <input
              id="faucet-address"
              name="address"
              inputMode="text"
              autoComplete="off"
              spellCheck={false}
              placeholder="40-character lowercase hex address"
              value={address}
              onChange={(event) => setAddress(event.target.value)}
              disabled={submitting}
            />
            <button type="submit" className="button" disabled={!canRequest}>
              {submitting ? "Requesting…" : `Request ${info.initial_grant_sudh} Test SUDH`}
            </button>
          </div>
        </form>
        {formError ? <p className="explorer-error" role="alert">{formError}</p> : null}
        {grant ? (
          <div className="notice" aria-live="polite">
            <strong>{grant.amount_sudh} Test SUDH submitted.</strong>
            <p>
              Status: {grant.status}. Transaction{" "}
              <Link className="text-link" href={`/explorer/tx?id=${encodeURIComponent(grant.transaction_id)}`}>
                {grant.transaction_id.slice(0, 12)}…
              </Link>
            </p>
          </div>
        ) : null}
      </section>
    </div>
  );
}
