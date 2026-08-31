import Link from "next/link";
import Image from "next/image";
import { PageHero } from "@/components/page-hero";

export default function AboutPage() {
  return (
    <div className="section-shell page-stack">
      <section className="hero-grid">
        <div>
          <PageHero
            eyebrow="ABOUT SUDHARMA"
            title="Built in India. Built to be open. Built for everyone."
            description="Sudharma Network is a student-built blockchain project created in India with a simple ambition: learn deeply, build openly and contribute something meaningful to decentralized technology."
          />
          <div className="hero-actions">
            <Link className="button primary" href="/downloads">Explore Downloads</Link>
            <Link className="button secondary" href="/developers">Build With Us</Link>
          </div>
        </div>
        <div className="hero-visual info-card" aria-label="Sudharma Network identity">
          <Image src="/sudharma-logo.png" alt="Sudharma Network logo" width={180} height={180} priority />
          <p className="mono">FROM INDIA · OPEN TO THE WORLD</p>
        </div>
      </section>

      <section className="info-card page-stack">
        <p className="eyebrow">OUR STORY</p>
        <h2>Curiosity became a network.</h2>
        <p>We are students from India working to turn curiosity, experimentation and engineering into a blockchain ecosystem that people can explore, test, improve and build upon.</p>
        <p>Sudharma began with the belief that meaningful technology does not have to start inside a large institution. It can begin with learners who are willing to study difficult problems, publish their work, accept feedback and keep improving what they build.</p>
        <p>Our goal is not to create a project that is useful in only one way. We want each part of Sudharma to give people a genuine reason to participate, whether they arrive as users, miners, developers or simply as people who want to learn.</p>
      </section>

      <section className="card-grid">
        <article className="info-card">
          <p className="eyebrow">USE</p>
          <h2>Use Sudharma</h2>
          <p>A wallet holder or testnet participant should find an accessible way to learn how the network works, send transactions, explore releases and participate without needing to be a protocol expert first.</p>
          <Link className="text-link" href="/wallet">Explore the wallet →</Link>
        </article>
        <article className="info-card">
          <p className="eyebrow">MINE</p>
          <h2>Mine Sudharma</h2>
          <p>Miners should be able to contribute computing power with practical GPU hardware, verify performance openly and help us improve mining compatibility across NVIDIA, AMD and other OpenCL-capable devices.</p>
          <Link className="text-link" href="/mining">Explore mining →</Link>
        </article>
        <article className="info-card">
          <p className="eyebrow">BUILD</p>
          <h2>Build on Sudharma</h2>
          <p>Developers should find source code, APIs, documentation and infrastructure that invite experimentation. Sudharma is open source because we want good ideas and careful engineering to come from beyond the original team.</p>
          <Link className="text-link" href="/developers">Developer resources →</Link>
        </article>
      </section>

      <section className="info-card page-stack">
        <p className="eyebrow">OPEN SOURCE · OPEN PARTICIPATION</p>
        <h2>Built in public, improved in public.</h2>
        <p>We are still students, and Sudharma is still growing. We see that as a strength. The project is being developed in public so mistakes can become improvements, experiments can become working technology and contributors from anywhere in the world can help shape what comes next.</p>
        <p>Holding or using SUDH, mining, running software or contributing code can each be a different way to participate in the same open ecosystem. Sudharma does not promise guaranteed financial returns, effortless profits or risk-free outcomes. Our responsibility is to build useful technology, communicate its current readiness clearly and keep improving the experience for people who choose to participate.</p>
      </section>

      <section className="info-card page-stack">
        <p className="eyebrow">WHY WE ARE BUILDING</p>
        <h2>More than another cryptocurrency.</h2>
        <p>Sudharma is an attempt to build an open technological foundation and discover how far determined learners, builders, miners, users and contributors can take it together. A useful network should reward curiosity with knowledge, participation with capability and contribution with visible progress.</p>
        <p>From India, open to the world.</p>
        <div className="hero-actions">
          <Link className="button primary" href="/testnet">Join the Testnet</Link>
          <a className="button secondary" href="https://github.com/sudharma-networks/sudharma">View GitHub ↗</a>
        </div>
      </section>
    </div>
  );
}
