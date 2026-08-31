import Link from "next/link";
import { PageHero } from "@/components/page-hero";
import { StatusChip } from "@/components/status-chip";
import { ReportProblemLink } from "@/components/report-problem-link";

const topics = [["NVIDIA", "nvidia", "CUDA GPU miner for Khushi Algorithm."], ["AMD / OpenCL", "amd", "OpenCL GPU miner for AMD and compatible cards."], ["Solo mining", "solo", "One-click GPU miner: paste a wallet address and start."], ["Pools", "pools", "Pool protocol and worker compatibility."], ["Kryptex", "kryptex", "Stratum/Kryptex compatibility work."], ["Benchmarks", "benchmarks", "Reproducible GPU performance reporting."], ["Troubleshooting", "troubleshooting", "Safe diagnostics for GPU miner issues."]] as const;

export default function MiningPage() {
  return <div className="section-shell page-stack"><PageHero eyebrow="MINING" title="Mine Sudharma on GPU only." description="Sudharma uses the Khushi GPU-PoW algorithm. There is no CPU miner and no ASIC miner — not on public testnet, and not on mainnet. NVIDIA CUDA and AMD/OpenCL GPUs are the supported hardware."><Link className="button" href="/downloads">GPU miner downloads</Link></PageHero>
    <section className="notice"><StatusChip status="Experimental" /><p>Public-testnet and mainnet mining are GPU-only. CPU mining and ASIC mining are not supported. Current Windows miner builds remain experimental until GPU-PoW work is issued by the live network.</p></section>
    <section className="info-card page-stack">
      <p className="eyebrow">ONE-CLICK WINDOWS GPU MINER</p>
      <h2>Paste your wallet address. Start mining.</h2>
      <p>The Windows miner asks only for the 40-character Sudharma address that should receive rewards. It never asks for a seed phrase or private key. Place the NVIDIA or AMD Khushi GPU hasher in the same folder, then double-click Start Mining.</p>
      <p>Sudharma does not ship a CPU miner and does not support ASIC firmware.</p>
    </section>
    <section className="card-grid">{topics.map(([title, slug, text]) => <Link className="info-card linked-card" href={`/mining/${slug}`} key={slug}><span className="mono">GUIDE</span><h2>{title}</h2><p>{text}</p><span className="text-link">Open guide →</span></Link>)}</section>
    <ReportProblemLink component="Mining" context="Hub" />
  </div>;
}
