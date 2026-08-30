import { FaucetStatus } from "@/components/faucet-status";
import { PageHero } from "@/components/page-hero";

export default function FaucetPage() {
  return (
    <div className="section-shell page-stack">
      <PageHero
        eyebrow="FAUCET"
        title="Public testnet SUDH distribution."
        description="The live faucet issues testnet-only SUDH through the reviewed public RPC proxy. Testnet coins have no mainnet value."
      />
      <FaucetStatus />
    </div>
  );
}
