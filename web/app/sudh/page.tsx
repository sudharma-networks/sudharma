import { PageHero } from "@/components/page-hero";
import { SUDH_PARAMETERS } from "@/lib/project";

const emission = [
  ["Year 1", "16%", "8.16M SUDH"],
  ["Year 2", "14%", "7.14M SUDH"],
  ["Year 3", "13%", "6.63M SUDH"],
  ["Year 4", "12%", "6.12M SUDH"],
  ["Year 5", "11%", "5.61M SUDH"],
  ["Year 6", "10%", "5.10M SUDH"],
  ["Year 7", "8%", "4.08M SUDH"],
  ["Year 8", "7%", "3.57M SUDH"],
  ["Year 9", "5%", "2.55M SUDH"],
  ["Year 10", "4%", "2.04M SUDH"]
] as const;

export default function SudhPage() {
  return <div className="section-shell page-stack">
    <PageHero eyebrow="SUDH / MAINNET TOKENOMICS" title="The native coin of Sudharma Network." description="SUDH is designed as a finite Proof-of-Work asset with transparent issuance, no premine and deterministic rules that can be verified in code." />

    <section className="notice"><strong>Approved Mainnet Design · Implementation in progress.</strong><p>This is the approved economic policy for mainnet. It is not the economics currently running on the public testnet, and no mainnet activation is being claimed here.</p></section>

    <section>
      <div className="section-heading"><p className="eyebrow">THE SUPPLY STORY</p><h2>51 Million. No Premine. Predictable Supply.</h2><p>The hard cap is 51,000,000 SUDH. Every newly issued coin is scheduled to enter through Proof-of-Work block subsidy; there is no founder or treasury premine in this design.</p></div>
      <div className="stats-grid">
        <div className="stat"><span>Maximum supply</span><strong>51,000,000 SUDH</strong></div>
        <div className="stat"><span>Premine</span><strong>0 SUDH</strong></div>
        <div className="stat"><span>Mining duration</span><strong>10-year mining era</strong></div>
        <div className="stat"><span>Reward transitions</span><strong>40 quarterly epochs</strong></div>
        <div className="stat"><span>Target block time</span><strong>60 seconds</strong></div>
        <div className="stat"><span>After final subsidy block</span><strong>0 new SUDH</strong></div>
      </div>
    </section>

    <section>
      <div className="section-heading"><p className="eyebrow">DECLINING EMISSION</p><h2>More issuance early. Less new supply over time.</h2><p>The ten annual targets add to exactly 51 million SUDH. Rewards step down through 40 quarterly epochs so miners see a gradual decline instead of a single sharp yearly cliff.</p></div>
      <div className="parameter-table">{emission.map(([year, percent, amount]) => <div key={year}><span>{year} · {percent} of hard cap</span><strong>{amount}</strong></div>)}</div>
      <p className="muted">Consensus uses block height, not calendar dates. The ten-year duration is nominal at the 60-second target block interval.</p>
    </section>

    <section className="split-section">
      <div><p className="eyebrow">TRANSACTION FEES</p><h2>Security and development are funded by network use.</h2></div>
      <div><p><strong>0.09% goes to miners</strong> from the transaction amount as the miner fee portion.</p><p><strong>0.01% goes to development treasury</strong> as the protocol development portion.</p><p>Total transaction fee: 0.10%. Transaction fees redistribute existing SUDH; they do not create new supply.</p></div>
    </section>

    <section>
      <div className="section-heading"><p className="eyebrow">AFTER THE MINING ERA</p><h2>The hard cap stays hard.</h2><p>After the final subsidy-bearing block, new block subsidy becomes zero. Under the approved v1 design there is no tail emission; miners then depend on transaction-fee revenue. Long-term fee-only security must be stress-tested before mainnet launch.</p></div>
    </section>

    <section className="notice"><strong>Scarcity is a protocol rule, not a price promise.</strong><p>A finite supply can reduce future issuance pressure, but market price still depends on demand, utility, liquidity, adoption, security and broader market conditions.</p></section>

    <section>
      <div className="section-heading"><p className="eyebrow">CURRENT PUBLIC TESTNET REFERENCE</p><h2>Testnet remains separate and unchanged.</h2><p>The values below describe the current development/testnet parameter set and remain subject to change before mainnet. They are shown so users can clearly distinguish today&apos;s test environment from the approved mainnet design above.</p></div>
      <div className="parameter-table">{SUDH_PARAMETERS.map(([label, value]) => <div key={label}><span>{label}</span><strong>{value}</strong></div>)}</div>
    </section>
  </div>;
}
