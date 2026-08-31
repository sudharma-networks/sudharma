import { PageHero } from "@/components/page-hero";
import { StatusChip } from "@/components/status-chip";

const phases = [
  ["Protocol + public testnet foundation", "Completed", "Core node, networking, RPC and initial public test infrastructure."],
  ["GPU-PoW mining", "In Development", "Khushi GPU-PoW only. CUDA/OpenCL miners; CPU and ASIC mining are not supported on testnet or mainnet."],
  ["Wallet + public web experience", "In Development", "Android wallet hardening, website, downloads and safe public integrations."],
  ["Explorer + faucet web integration", "Testnet", "Read-only explorer and public faucet request flow are live against the testnet wallet proxy."],
  ["Developer platform", "Planned", "Stable public APIs, SDKs and higher-level application capabilities."],
  ["Mainnet readiness", "Planned", "Security review, release hardening, operational readiness and explicit launch decision."]
] as const;

export default function RoadmapPage() { return <div className="section-shell page-stack"><PageHero eyebrow="ROADMAP" title="Progress without pretending the future is finished." description="Roadmap labels describe current project state: Completed, In Development, or Planned." /><section className="timeline">{phases.map(([title, status, text], i) => <article key={title}><span className="timeline-index">0{i + 1}</span><div><StatusChip status={status} /><h2>{title}</h2><p>{text}</p></div></article>)}</section></div>; }
