import { ExplorerDashboard } from "@/components/explorer-dashboard";
import { PageHero } from "@/components/page-hero";
import { resolveExplorerAPIBaseURL } from "@/lib/explorer-config";

export default function ExplorerPage() {
  const apiBaseUrl = resolveExplorerAPIBaseURL(process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL);

  return (
    <div className="section-shell page-stack">
      <PageHero
        eyebrow="BLOCKCHAIN EXPLORER"
        title="Follow Sudharma testnet in real time."
        description="Etherscan-style read-only visibility: network status, latest blocks, confirmed and pending transactions, address history, and unified search. Data is fetched live from Seed-1 and Seed-2 nodes with mempool and miner integration."
      />
      <ExplorerDashboard apiBaseUrl={apiBaseUrl} />
    </div>
  );
}
