import Link from "next/link";
import { notFound } from "next/navigation";
import { PageHero } from "@/components/page-hero";
import { StatusChip } from "@/components/status-chip";
import { ReportProblemLink } from "@/components/report-problem-link";

const TOPICS = {
  nvidia: { title: "NVIDIA GPU Mining", status: "Experimental" as const, intro: "Sudharma is GPU-only. The NVIDIA path uses CUDA for Khushi GPU-PoW. CPU mining and ASIC mining are not supported on public testnet or mainnet.", points: ["Use current NVIDIA drivers and a verified Khushi CUDA miner build.", "Paste only your 40-character wallet address. Never enter a seed phrase.", "Start on public-testnet. Mainnet mining stays closed until launch and will still be GPU-only."] },
  amd: { title: "AMD / OpenCL Mining", status: "Experimental" as const, intro: "The OpenCL path targets AMD and other compatible GPUs. Sudharma has no CPU miner and no ASIC miner.", points: ["Confirm an OpenCL GPU runtime before launching the miner.", "Treat 4 GB support as a project target that still requires per-device validation.", "Report GPU model, driver and public error text without credentials."] },
  solo: { title: "Solo Mining", status: "In Development" as const, intro: "Solo mining is the one-click GPU miner: paste a reward address and connect to public-testnet. Mainnet uses the same GPU-only rule when it opens.", points: ["Use the official Windows GPU miner. It asks only for a wallet address.", "Keep node administration interfaces private.", "CPU and ASIC backends are rejected by the miner and by the protocol policy."] },
  pools: { title: "Pool Mining", status: "In Development" as const, intro: "Pool compatibility is being built around clear worker identity and job contracts.", points: ["Pool software and network contracts are not yet a production promise.", "Worker accounting should never require wallet private keys.", "Verified pool onboarding guidance will be versioned with releases."] },
  kryptex: { title: "Kryptex Compatibility", status: "In Development" as const, intro: "Stratum/Kryptex compatibility is an active development track.", points: ["Worker identity, job delivery and share validation are being tested in stages.", "Compatibility status can change as contracts are hardened.", "Do not use unofficial connection parameters represented as production settings."] },
  benchmarks: { title: "Mining Benchmarks", status: "Planned" as const, intro: "A reproducible benchmark catalog will follow stable miner builds.", points: ["Benchmarks must include GPU model, VRAM, driver, miner version and settings.", "Efficiency and stability matter alongside hashrate.", "No speculative earnings or price assumptions will be published here."] },
  troubleshooting: { title: "Mining Troubleshooting", status: "In Development" as const, intro: "Use structured diagnostics to make mining reports useful.", points: ["Capture miner version, GPU model, VRAM, driver and public error text.", "Never include private keys, seed phrases or privileged RPC credentials.", "Confirm you are using an official repository/release artifact before debugging."] }
} as const;

type Topic = keyof typeof TOPICS;
export function generateStaticParams() { return Object.keys(TOPICS).map((topic) => ({ topic })); }

export default async function MiningTopicPage({ params }: { params: Promise<{ topic: string }> }) {
  const { topic } = await params;
  if (!(topic in TOPICS)) notFound();
  const item = TOPICS[topic as Topic];
  return <div className="section-shell page-stack"><PageHero eyebrow="MINING GUIDE" title={item.title} description={item.intro}><Link className="button" href="/downloads">Downloads</Link><Link className="button secondary" href="/mining">All mining guides</Link></PageHero><StatusChip status={item.status} /><section className="steps">{item.points.map((point, i) => <article key={point}><span className="mono">0{i + 1}</span><p>{point}</p></article>)}</section><ReportProblemLink component="Mining" context={item.title} /></div>;
}
