"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { resolveExplorerAPIBaseURL } from "@/lib/explorer-config";

type FaucetInfo = {
  enabled: boolean;
  challenge_address: string;
  initial_grant_sudh: number;
  challenge_send_sudh: number;
  challenge_reward_sudh: number;
  max_rounds: number;
  cooldown_hours: number;
  testnet_only: boolean;
};

export function FaucetStatus() {
  const [info, setInfo] = useState<FaucetInfo | null>(null);
  const [error, setError] = useState<string | null>(null);
  const apiBase = resolveExplorerAPIBaseURL(process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL);

  useEffect(() => {
    let cancelled = false;
    fetch(`${apiBase}/v1/faucet/info`, { cache: "no-store" })
      .then(async (response) => {
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return response.json() as Promise<FaucetInfo>;
      })
      .then((body) => {
        if (!cancelled) setInfo(body);
      })
      .catch((fetchError) => {
        if (!cancelled) setError(String(fetchError?.message || fetchError));
      });
    return () => {
      cancelled = true;
    };
  }, [apiBase]);

  if (error) {
    return <p className="notice">Public faucet status is temporarily unavailable ({error}).</p>;
  }

  if (!info) {
    return <p className="notice">Loading public faucet status…</p>;
  }

  return (
    <section className="status-grid">
      <article>
        <span className={`status-chip ${info.enabled ? "live" : "planned"}`}>{info.enabled ? "Live" : "Unavailable"}</span>
        <h2>Initial test grant</h2>
        <p>Request {info.initial_grant_sudh} testnet SUDH once per address through the public API.</p>
        <code>POST {apiBase}/v1/faucet/request</code>
      </article>
      <article>
        <span className="status-chip live">Challenge</span>
        <h2>Send {info.challenge_send_sudh}, receive {info.challenge_reward_sudh}</h2>
        <p>After the initial grant confirms, send exactly {info.challenge_send_sudh} SUDH to the challenge address, then claim the reward.</p>
        <code>{info.challenge_address}</code>
        <p><code>POST {apiBase}/v1/faucet/challenge</code></p>
      </article>
      <article>
        <span className="status-chip testnet">Testnet only</span>
        <h2>Safe public boundary</h2>
        <p>The website does not hold faucet signing keys. Wallet apps and scripts call the reviewed public RPC proxy directly.</p>
        <div className="hero-actions">
          <Link className="button" href="/explorer">Open Explorer</Link>
          <Link className="button secondary" href="/testnet">Testnet guide</Link>
        </div>
      </article>
    </section>
  );
}
