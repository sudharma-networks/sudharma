import Link from "next/link";
import { PageHero } from "@/components/page-hero";
import { StatusChip } from "@/components/status-chip";
import { ReportProblemLink } from "@/components/report-problem-link";

export default function WalletPage() {
  return <div className="section-shell page-stack"><PageHero eyebrow="WALLET" title="Use Sudharma without hiding the development state." description="Create, receive and send on the public testnet while wallet software is still being hardened for a future mainnet release."><Link className="button" href="/downloads">Wallet downloads</Link><Link className="button secondary" href="/testnet">Testnet</Link></PageHero>
    <section className="card-grid"><article className="info-card"><StatusChip status="Testnet" /><h2>Android wallet</h2><p>Designed to connect to Sudharma public test infrastructure without asking everyday users to enter privileged node administration endpoints.</p></article><article className="info-card"><StatusChip status="In Development" /><h2>CLI + advanced flows</h2><p>Command-line and operator-oriented wallet workflows remain development tooling until release packaging is verified.</p></article><article className="info-card"><h2>Create · Receive · Send</h2><p>Generate a wallet, protect recovery material, receive test SUDH to an address, and confirm destinations before sending.</p></article></section>
    <section className="notice danger"><strong>Protect your recovery material.</strong><p>Never paste a seed phrase or private key into this website, a support report, a chat message or a download page. The website will never ask for it.</p></section>
    <section className="split-section"><div><p className="eyebrow">TEST FIRST</p><h2>Use testnet before anything else.</h2></div><div><p>Testnet coins have no mainnet value. Use them to learn wallet behavior and report reproducible issues.</p><div className="hero-actions"><Link className="button secondary" href="/faucet">Request test SUDH</Link><ReportProblemLink component="Wallet" context="General" /></div></div></section>
  </div>;
}
