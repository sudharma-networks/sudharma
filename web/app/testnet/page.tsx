import { PageHero } from "@/components/page-hero";
import { StatusChip } from "@/components/status-chip";
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
          <StatusChip status="Testnet" />
          <h2>Wallet testing</h2>
          <p>Testnet SUDH exists for functional testing only and has no mainnet value.</p>
        </article>
        <article>
          <StatusChip status="Testnet" />
          <h2>Live web explorer</h2>
          <p>Read-only block, transaction, address and mempool views are available through the public explorer.</p>
        </article>
        <article>
          <StatusChip status="Testnet" />
          <h2>Public faucet</h2>
          <p>Request an initial testnet grant from the faucet page. Challenge rounds remain in the Android wallet.</p>
        </article>
      </section>
    </div>
  );
}
