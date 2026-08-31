import Link from "next/link";
import { PageHero } from "@/components/page-hero";
import { StatusChip } from "@/components/status-chip";

const layers = ["Wallet", "Transaction", "Nodes", "Mempool", "Miner", "Block", "Network"];
const topics = [
  ["Proof of Work", "Miners compete to extend the chain with valid work; nodes independently verify every accepted block."],
  ["Cumulative work", "Chain selection follows validated cumulative work rather than trusting a coordinator."],
  ["Peer-to-peer", "Nodes exchange chain and transaction data over the Sudharma P2P network."],
  ["Blocks + transactions", "Transactions enter node mempools, miners assemble candidates, and valid blocks advance the chain."],
  ["Sync + reorg", "Nodes synchronize from peers and can reorganize to a stronger valid chain when required."],
  ["Security hardening", "Protocol, wallet, mining and public-testnet paths remain under active pre-mainnet hardening."]
];

export default function NetworkPage() {
  return <div className="section-shell page-stack"><PageHero eyebrow="NETWORK" title="A Proof-of-Work network built to be inspected." description="Sudharma combines independently validating nodes, peer-to-peer synchronization, transaction relay, mining and cumulative-work chain selection." />
    <section><div className="section-heading"><p className="eyebrow">TRANSACTION FLOW</p><h2>From intent to shared state.</h2></div><div className="flow-strip">{layers.map((layer, i) => <span key={layer}>{layer}{i < layers.length - 1 ? <b aria-hidden="true">→</b> : null}</span>)}</div></section>
    <section className="card-grid">{topics.map(([title, text], i) => <article className="info-card" key={title}><span className="mono">0{i + 1}</span><h2>{title}</h2><p>{text}</p></article>)}</section>
    <section className="trust-band"><div><StatusChip status="In Development" /><h2>Run a node</h2></div><div><p>Public node documentation is being consolidated for pre-mainnet operators. Build-from-source guidance remains available through the repository.</p><Link className="button" href="/developers/node">Node guide</Link></div></section>
  </div>;
}
