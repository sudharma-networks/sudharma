import Link from "next/link";
import { notFound } from "next/navigation";
import { PageHero } from "@/components/page-hero";
import { StatusChip } from "@/components/status-chip";
import { ReportProblemLink } from "@/components/report-problem-link";

const TOPICS = {
  nvidia: { title: "NVIDIA GPU Mining", status: "Experimental" as const, intro: "Sudharma is GPU-only. The NVIDIA path uses CUDA for Khushi GPU-PoW. CPU mining and ASIC mining are not supported on public testnet or mainnet.", points: ["Use current NVIDIA drivers and a verified Khushi CUDA miner build.", "Paste only your 40-character wallet address. Never enter a seed phrase.", "Start on public-testnet. Mainnet mining stays closed until launch and will still be GPU-only."] },
  amd: { title: "AMD / OpenCL Mining", status: "Experimental" as const, intro: "The OpenCL path targets AMD and other compatible GPUs. Sudharma has no CPU miner and no ASIC miner.", points: ["Confirm an OpenCL GPU runtime before launching the miner.", "Treat 4 GB support as a project target that still requires per-device validation.", "Report GPU model, driver and public error text without credentials."] },
  solo: { title: "Solo Mining", status: "Experimental" as const, intro: "Solo mining is the one-click GPU miner: paste a reward address and connect to public-testnet. You keep the full block reward when your GPU finds a block.", points: ["Use the official Windows GPU miner or sudharma-miner CLI.", "Work comes from POST /v1/mining/work; submit solved blocks to /v1/mining/submit.", "CPU and ASIC backends are rejected by the miner and by the protocol policy."] },
  pools: { title: "Pool Mining", status: "Experimental" as const, intro: "Pool operators can run the reference sudharma-pool server with PPS, PPLNS, SOLO, or FPPS payout modes. Workers connect over Stratum v1 using wallet.worker logins.", points: ["Reference server: cmd/sudharma-pool with deployment/testnet/pool.example.json.", "Pool difficulty is lower than network difficulty; valid shares accumulate payouts.", "Workers never send private keys — only wallet.worker identity and nonces."] },
  kryptex: { title: "Kryptex / Stratum Compatibility", status: "Experimental" as const, intro: "Sudharma ships a Stratum v1 bridge for pool operators. mining.notify delivers candidate block fields plus pool target; mining.submit accepts [login, job_id, nonce].", points: ["Login format: 40-hex-wallet.worker-name (same as RVN/BTC pools).", "See docs/audits/2026-08-31-pool-mining-architecture.md for the full contract.", "Do not use unofficial connection parameters represented as production settings."] },
  benchmarks: { title: "Mining Benchmarks", status: "Planned" as const, intro: "A reproducible benchmark catalog will follow stable miner builds.", points: ["Benchmarks must include GPU model, VRAM, driver, miner version and settings.", "Efficiency and stability matter alongside hashrate.", "No speculative earnings or price assumptions will be published here."] },
  troubleshooting: { title: "Mining Troubleshooting", status: "Experimental" as const, intro: "Most GPU miner failures are connectivity, address format, or backend policy issues. Use this checklist before opening a report.", points: ["404 on /v1/mining/work: mining routes are not enabled on that RPC endpoint — use the public testnet proxy or seed nginx allowlist.", "GPU work fetch failed: confirm HTTPS proxy URL, wallet address is 40 lowercase hex, and you are on public-testnet.", "Share not accepted / stale job: pool job expired — reconnect Stratum or restart sudharma-miner.", "CPU or ASIC rejected: Sudharma is GPU-only (sudharma-gpupow-v1); install Khushi CUDA/OpenCL binaries for hashing.", "Never send seed phrases, private keys, or admin RPC credentials in bug reports."] },
} as const;

type Topic = keyof typeof TOPICS;
export function generateStaticParams() { return Object.keys(TOPICS).map((topic) => ({ topic })); }

export default async function MiningTopicPage({ params }: { params: Promise<{ topic: string }> }) {
  const { topic } = await params;
  if (!(topic in TOPICS)) notFound();
  const item = TOPICS[topic as Topic];
  return <div className="section-shell page-stack"><PageHero eyebrow="MINING GUIDE" title={item.title} description={item.intro}><Link className="button" href="/downloads">Downloads</Link><Link className="button secondary" href="/mining">All mining guides</Link></PageHero><StatusChip status={item.status} /><section className="steps">{item.points.map((point, i) => <article key={point}><span className="mono">0{i + 1}</span><p>{point}</p></article>)}</section><ReportProblemLink component="Mining" context={item.title} /></div>;
}
