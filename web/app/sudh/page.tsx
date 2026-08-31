import { PageHero } from "@/components/page-hero";
import { SUDH_PARAMETERS } from "@/lib/project";

export default function SudhPage() {
  return <div className="section-shell page-stack"><PageHero eyebrow="SUDH" title="The native coin of Sudharma Network." description="SUDH is the protocol-native unit used for block rewards, transaction fees and value transfer on Sudharma." />
    <section className="notice"><strong>Pre-mainnet parameter notice.</strong><p>These are the currently documented development parameters and are subject to change before mainnet through reviewed protocol updates.</p></section>
    <section><div className="section-heading"><p className="eyebrow">CURRENT PARAMETERS</p><h2>Simple, finite, visible.</h2></div><div className="parameter-table">{SUDH_PARAMETERS.map(([label, value]) => <div key={label}><span>{label}</span><strong>{value}</strong></div>)}</div></section>
    <section className="split-section"><div><p className="eyebrow">FEE SPLIT</p><h2>0.10% total transaction fee.</h2></div><div><p>The current design allocates 0.09% to miners and 0.01% to development. These values are protocol development parameters, not a promise of future economics.</p></div></section>
  </div>;
}
