import Link from "next/link";
import { notFound } from "next/navigation";
import { PageHero } from "@/components/page-hero";
import { StatusChip } from "@/components/status-chip";

const TOPICS = {
  "getting-started": { title: "Getting Started", text: "Sudharma is developed in the public repository. Start by reading repository documentation, building with the supported Go toolchain, and running tests before changing protocol behavior." },
  node: { title: "Node Development", text: "Sudharma nodes validate chain state, communicate over P2P, expose scoped RPC interfaces and persist local data. Keep administration endpoints private and separate from public web APIs." },
  rpc: { title: "RPC / API", text: "RPC is an integration boundary, not a reason to expose privileged node controls to browsers. Public website integrations will use explicitly safe read-only contracts." },
  wallets: { title: "Wallet Integration", text: "Wallet integrations should isolate signing material from public services, validate destinations and never transmit seed phrases or private keys to website infrastructure." },
  payments: { title: "Payment Integration", text: "Payment applications should create and track transactions through documented public interfaces and clearly distinguish testnet state from future mainnet state." },
  contributing: { title: "Contributing", text: "Use focused changes, tests and review. Security-sensitive reports belong in the repository security process rather than public issue content when disclosure could create risk." },
  protocol: { title: "Protocol", text: "The protocol combines Proof of Work, block and transaction validation, cumulative-work chain selection, peer synchronization, mempool handling and deterministic state transitions." }
} as const;
type Topic = keyof typeof TOPICS;
export function generateStaticParams() { return Object.keys(TOPICS).map((topic) => ({ topic })); }

export default async function DeveloperTopicPage({ params }: { params: Promise<{ topic: string }> }) {
  const { topic } = await params;
  if (!(topic in TOPICS)) notFound();
  const item = TOPICS[topic as Topic];
  return <div className="section-shell page-stack"><PageHero eyebrow="DEVELOPER GUIDE" title={item.title} description={item.text}><a className="button" href="https://github.com/sudharma-networks/sudharma">Repository</a><Link className="button secondary" href="/developers">Developer hub</Link></PageHero><section className="notice"><StatusChip status="In Development" /><p>Interfaces may change before mainnet. Pin integration work to reviewed versions and verify source before deployment.</p></section></div>;
}
