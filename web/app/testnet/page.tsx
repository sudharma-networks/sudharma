import Link from "next/link";
import { PageHero } from "@/components/page-hero";
import { StatusChip } from "@/components/status-chip";
import { PUBLIC_EXPLORER_API_BASE_URL } from "@/lib/explorer-config";

export default function TestnetPage() {
  return (
    <div className="section-shell page-stack">
      <PageHero
        eyebrow="TESTNET"
        title="A public place to test before mainnet."
        description="Sudharma public testnet is used to exercise node, wallet, transaction, mining and infrastructure behavior before production launch."
      />
      <section className="status-grid">
        <article>
          <StatusChip status="Testnet" />
          <h2>Network testing</h2>
          <p>Public seed infrastructure supports ongoing connectivity and protocol testing.</p>
        </article>
        <article>
          <StatusChip status="Live" />
          <h2>Blockchain explorer</h2>
          <p>Read-only blocks, transactions, addresses, and search are available on the public website explorer and API.</p>
          <Link className="text-link" href="/explorer">Open explorer →</Link>
        </article>
        <article>
          <StatusChip status="Live" />
          <h2>Faucet + wallet flow</h2>
          <p>Request 100 test SUDH, send 25 SUDH to the challenge address, and claim 50 SUDH through the public faucet API when enabled.</p>
          <Link className="text-link" href="/faucet">Faucet details →</Link>
        </article>
      </section>
      <section className="notice">
        <p>Public API base URL: <code>{PUBLIC_EXPLORER_API_BASE_URL}</code></p>
      </section>
    </div>
  );
}
