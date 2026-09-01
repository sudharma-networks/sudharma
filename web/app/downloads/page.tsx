import { PageHero } from "@/components/page-hero";
import { DownloadCard } from "@/components/download-card";
import { DOWNLOADS, type DownloadKind } from "@/lib/downloads";

const sections: [DownloadKind, string][] = [["wallet", "Wallets"], ["miner", "GPU Miners"], ["node", "Node Software"], ["source", "Source Code"], ["developer", "Developer Resources"]];

export default function DownloadsPage() {
  return <div className="section-shell page-stack"><PageHero eyebrow="DOWNLOADS" title="Verified software, explicit status." description="Download controls appear only where a public artifact URL and provenance are verified. Development items remain clearly unavailable until they are ready." />
    <section className="notice danger"><strong>Download safely.</strong><p>Use official links only. Verify checksums when published. Never enter a seed phrase or private key on a download page, support form or website prompt. Sudharma miners are GPU-only; CPU and ASIC mining are not supported.</p></section>
    {sections.map(([kind, label]) => <section key={kind}><div className="section-heading"><p className="eyebrow">{label.toUpperCase()}</p><h2>{label}</h2></div><div className="download-grid">{DOWNLOADS.filter((item) => item.kind === kind).map((artifact) => <DownloadCard artifact={artifact} key={artifact.id} />)}</div></section>)}
    <section className="split-section"><div><p className="eyebrow">OPEN SOURCE</p><h2>Apache License 2.0</h2></div><div><p>Sudharma source is published openly. Read the repository license directly rather than relying on a website summary of its legal terms.</p><a className="text-link" href="https://github.com/sudharma-networks/sudharma/blob/main/LICENSE">Read LICENSE ↗</a></div></section>
  </div>;
}
