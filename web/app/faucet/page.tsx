import { PageHero } from "@/components/page-hero";
import { FaucetPanel } from "@/components/faucet-panel";
import { StatusChip } from "@/components/status-chip";
import { resolveFaucetAPIBaseURL } from "@/lib/faucet-config";

export default function FaucetPage() {
  const apiBaseUrl = resolveFaucetAPIBaseURL(process.env.NEXT_PUBLIC_FAUCET_API_BASE_URL);

  return (
    <div className="section-shell page-stack">
      <PageHero
        eyebrow="FAUCET"
        title="Request testnet SUDH safely."
        description="The public faucet issues testnet-only SUDH through the reviewed wallet proxy. Test coins have no mainnet value."
      >
        <StatusChip status="Testnet" />
      </PageHero>
      <FaucetPanel apiBaseUrl={apiBaseUrl} />
    </div>
  );
}
