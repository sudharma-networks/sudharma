import { ExplorerDashboard } from "@/components/explorer-dashboard";
import { PageHero } from "@/components/page-hero";

export default function ExplorerPage() {
  const apiBaseUrl = process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL ?? "";

  return (
    <div className="section-shell page-stack">
      <PageHero
        eyebrow="BLOCKCHAIN EXPLORER"
        title="Follow Sudharma testnet in real time."
        description="Read-only visibility into the canonical pre-mainnet chain: network status, recent blocks, confirmed transactions, addresses, and search. No placeholder chain statistics are shown."
      />
      <ExplorerDashboard apiBaseUrl={apiBaseUrl} />
    </div>
  );
}
