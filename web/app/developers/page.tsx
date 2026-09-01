import Link from "next/link";
import { PageHero } from "@/components/page-hero";
import { StatusChip } from "@/components/status-chip";
import { ReportProblemLink } from "@/components/report-problem-link";

const topics = [["Getting started", "getting-started", "Build the project and understand repository boundaries."], ["Run a node", "node", "Node configuration, sync and operator concepts."], ["RPC / API", "rpc", "Public integration surface and safe endpoint boundaries."], ["GPU mining", "mining", "Solo HTTP mining API, Stratum pools, and worker integration."], ["Wallet integration", "wallets", "Wallet-oriented application concepts."], ["Payments", "payments", "Transaction integration principles."], ["Contributing", "contributing", "Issues, changes, tests and security reporting."], ["Protocol", "protocol", "Consensus and network architecture overview."]] as const;

export default function DevelopersPage() {
  return <div className="section-shell page-stack"><PageHero eyebrow="DEVELOPERS" title="Build on Sudharma in the open." description="Start from source, run nodes, study interfaces and contribute against explicit pre-mainnet boundaries."><a className="button" href="https://github.com/sudharma-networks/sudharma">GitHub</a><Link className="button secondary" href="/downloads">Developer downloads</Link></PageHero>
    <section className="card-grid">{topics.map(([title, slug, text]) => <Link className="info-card linked-card" href={`/developers/${slug}`} key={slug}><span className="mono">DOC</span><h2>{title}</h2><p>{text}</p><span className="text-link">Open →</span></Link>)}</section>
    <section className="status-grid"><article><StatusChip status="Planned" /><h3>SDKs</h3><p>Language-specific client SDKs follow stable public interfaces.</p></article><article><StatusChip status="Planned" /><h3>Tokens + smart contracts</h3><p>Application-layer standards remain planned capabilities, not current production features.</p></article></section>
    <ReportProblemLink component="Developers" context="Hub" />
  </div>;
}
