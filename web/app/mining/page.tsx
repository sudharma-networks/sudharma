import Link from "next/link";
import { PageHero } from "@/components/page-hero";
import { StatusChip } from "@/components/status-chip";
import { ReportProblemLink } from "@/components/report-problem-link";

const topics = [["NVIDIA", "nvidia", "CUDA miner development and compatibility."], ["AMD / OpenCL", "amd", "Cross-vendor OpenCL mining path."], ["Solo mining", "solo", "Direct node-oriented mining workflow."], ["Pools", "pools", "Pool protocol and worker compatibility."], ["Kryptex", "kryptex", "Stratum/Kryptex compatibility work."], ["Benchmarks", "benchmarks", "Reproducible performance reporting."], ["Troubleshooting", "troubleshooting", "Safe diagnostics for common miner issues."]] as const;

export default function MiningPage() {
  return <div className="section-shell page-stack"><PageHero eyebrow="MINING" title="Mine Sudharma across modern GPUs." description="Sudharma GPU-PoW development targets practical mining on supported 4 GB-and-above GPUs across NVIDIA and AMD/OpenCL paths, subject to hardware and driver validation."><Link className="button" href="/downloads">Miner downloads</Link></PageHero>
    <section className="notice"><StatusChip status="Experimental" /><p>GPU mining remains staged pre-mainnet work. Do not treat current software or benchmark expectations as a production-mainnet guarantee.</p></section>
    <section className="card-grid">{topics.map(([title, slug, text]) => <Link className="info-card linked-card" href={`/mining/${slug}`} key={slug}><span className="mono">GUIDE</span><h2>{title}</h2><p>{text}</p><span className="text-link">Open guide →</span></Link>)}</section>
    <ReportProblemLink component="Mining" context="Hub" />
  </div>;
}
