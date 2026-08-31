import Link from "next/link";
import { EcosystemCard } from "@/components/ecosystem-card";
import { ProjectActivity } from "@/components/project-activity";
import { ReadinessBadge } from "@/components/readiness-badge";
import { StatusChip } from "@/components/status-chip";
import { VisitorCounter } from "@/components/visitor-counter";
import visitorCounterConfig from "@/public/data/visitor-counter.json";

export default function Home() {
  return (
    <>
      <section className="home-hero section-shell">
        <div className="hero-grid">
          <div>
            <ReadinessBadge />
            <p className="eyebrow">SUDHARMA NETWORK / OPEN PROTOCOL</p>
            <h1>Open Blockchain.<br />Open Development.<br /><span>Built for Everyone.</span></h1>
            <p className="lead">Proof-of-Work infrastructure for users, miners, developers, researchers, and builders — developed openly from protocol to applications.</p>
            <div className="hero-actions"><Link className="button" href="/testnet">Explore Testnet</Link><Link className="button secondary" href="/downloads">Downloads</Link></div>
          </div>
          <div className="network-orbit" aria-label="Sudharma ecosystem diagram">
            <div className="orbit-core"><img src="/sudharma-logo.png" alt="Sudharma logo" width="108" height="108" /><strong>SUDH</strong><span>Proof of Work</span></div>
            <span className="orbit-label orbit-one">Wallets</span><span className="orbit-label orbit-two">Miners</span><span className="orbit-label orbit-three">Nodes</span><span className="orbit-label orbit-four">Builders</span>
          </div>
        </div>
      </section>

      <section className="section-shell section-block">
        <div className="section-heading"><p className="eyebrow">THREE WAYS IN</p><h2>One network. Different paths.</h2><p>Start with what you want to do. Every path leads back to transparent, open-source infrastructure.</p></div>
        <div className="ecosystem-grid">
          <EcosystemCard number="01" title="Use Sudharma" description="Create a wallet, transact on testnet, and learn how SUDH works." href="/wallet" links="Wallet · Transactions · Testnet · Faucet" />
          <EcosystemCard number="02" title="Mine Sudharma" description="Follow GPU mining development across NVIDIA, AMD/OpenCL, solo and pool paths." href="/mining" links="GPU Mining · NVIDIA · AMD · Guides" />
          <EcosystemCard number="03" title="Build on Sudharma" description="Run a node, inspect RPC interfaces, contribute code, and prepare integrations." href="/developers" links="Open Source · APIs · Protocol · Contributions" />
        </div>
      </section>

      <section className="section-shell split-section">
        <div><p className="eyebrow">MORE THAN A COIN</p><h2>Protocol, infrastructure, and an open development surface.</h2></div>
        <div className="feature-list"><p>Sudharma is being built as an open Proof-of-Work network with peer-to-peer nodes, wallets, GPU mining, public test infrastructure, APIs and future application tooling.</p><p>No fabricated live counters. No hidden mainnet claims. Development status is shown where it matters.</p></div>
      </section>

      <section className="section-shell section-block">
        <div className="section-heading"><p className="eyebrow">APPROVED MAINNET ECONOMICS</p><h2>Designed for Scarcity.</h2><p>Sudharma&apos;s approved mainnet design fixes issuance in advance: a finite hard cap, no premine and a gradually declining Proof-of-Work reward over a nominal ten-year mining era. Implementation is in progress; the public testnet keeps its current economics until mainnet activation.</p></div>
        <div className="stats-grid">
          <div className="stat"><span>Maximum supply</span><strong>51M</strong></div>
          <div className="stat"><span>Launch allocation</span><strong>0 Premine</strong></div>
          <div className="stat"><span>Mining era</span><strong>~10 Years</strong></div>
          <div className="stat"><span>Reward path</span><strong>40 Quarterly Epochs</strong></div>
          <div className="stat"><span>Target block time</span><strong>60 Seconds</strong></div>
          <div className="stat"><span>After final emission</span><strong>0 New Subsidy</strong></div>
        </div>
        <p className="muted">New issuance is designed to decline over time. Scarcity is a supply rule, not a promise of market price or investment return.</p>
        <Link className="text-link" href="/sudh">Explore tokenomics →</Link>
      </section>

      <section className="section-shell section-block">
        <div className="section-heading"><p className="eyebrow">PUBLIC REACH</p><h2>See the community discover Sudharma.</h2><p>The public counter is privacy-friendly and records at most one visit per browser/device each day. It does not collect names, wallet addresses, seed phrases or account details.</p></div>
        <div className="stats-grid"><VisitorCounter endpoint={visitorCounterConfig.endpoint} /></div>
      </section>

      <section className="section-shell section-block">
        <div className="section-heading"><p className="eyebrow">BUILD STATUS</p><h2>What exists now, and what comes next.</h2></div>
        <div className="status-grid">
          <article><StatusChip status="Testnet" /><h3>Public test infrastructure</h3><p>Nodes, wallet connectivity and challenge infrastructure are being exercised before mainnet.</p></article>
          <article><StatusChip status="Experimental" /><h3>GPU mining</h3><p>NVIDIA CUDA and AMD/OpenCL paths are under staged validation and compatibility work.</p></article>
          <article><StatusChip status="Testnet" /><h3>Explorer + faucet web</h3><p>Read-only explorer and public faucet request flow are live against the testnet wallet proxy.</p></article>
          <article><StatusChip status="Planned" /><h3>SDKs + application layer</h3><p>Developer SDKs, token standards and higher-level application capabilities follow protocol hardening.</p></article>
        </div>
      </section>

      <ProjectActivity />

      <section className="section-shell trust-band">
        <div><p className="eyebrow">SECURITY + TRANSPARENCY</p><h2>Verify, don’t assume.</h2></div>
        <div><p>Sudharma is open source under Apache-2.0. Security hardening is ongoing and there has not yet been an independent production audit.</p><div className="hero-actions"><a className="button secondary" href="https://github.com/sudharma-networks/sudharma">View source</a><Link className="button secondary" href="/roadmap">Roadmap</Link></div></div>
      </section>

      <section className="section-shell cta-grid"><Link href="/downloads"><span className="eyebrow">DOWNLOADS</span><h2>Verified software lives here.</h2><p>See status, platform and provenance before you download.</p></Link><Link href="/community"><span className="eyebrow">COMMUNITY</span><h2>Build in public with us.</h2><p>Repository, contribution and security channels.</p></Link><Link href="/support"><span className="eyebrow">SUPPORT</span><h2>Found a problem?</h2><p>Use the support entry point and safe reporting guidance.</p></Link></section>
    </>
  );
}
