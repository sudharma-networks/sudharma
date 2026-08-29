import { PageHero } from "@/components/page-hero";

export default function CommunityPage() {
  return (
    <div className="section-shell page-stack">
      <PageHero
        eyebrow="COMMUNITY"
        title="Open development starts with visible work."
        description="Follow development, meet public testnet participants and use Sudharma's official channels for announcements, mining tests, wallet feedback and open-source contributions."
      />

      <section className="info-card page-stack">
        <p className="eyebrow">OFFICIAL COMMUNITY</p>
        <h2>Join Sudharma in the open.</h2>
        <p>Use these official destinations to follow project announcements or join discussions with miners, testers and developers.</p>
        <p><strong>Safety:</strong> Only trust community links published on this website and never share a wallet seed phrase, private key, password or money with someone claiming to represent Sudharma.</p>
      </section>

      <section className="card-grid">
        <a className="info-card linked-card" href="https://t.me/sudharmanetworks">
          <p className="eyebrow">TELEGRAM · OFFICIAL</p>
          <h2>Official Telegram announcements</h2>
          <p>@sudharmanetworks — releases, testnet milestones, mining updates and important project notices.</p>
          <span className="text-link">Follow announcements ↗</span>
        </a>
        <a className="info-card linked-card" href="https://t.me/sudharma_community">
          <p className="eyebrow">TELEGRAM · COMMUNITY</p>
          <h2>Sudharma community discussion</h2>
          <p>@sudharma_community — discuss the public testnet, wallet, Khushi GPU mining, benchmarks, bugs and development.</p>
          <span className="text-link">Join the community ↗</span>
        </a>
        <a className="info-card linked-card" href="https://github.com/sudharma-networks/sudharma">
          <p className="eyebrow">SOURCE OF TRUTH</p>
          <h2>GitHub repository</h2>
          <p>Source, history, issues, pull requests, verified releases and project documentation.</p>
          <span className="text-link">Open GitHub ↗</span>
        </a>
        <a className="info-card linked-card" href="https://github.com/sudharma-networks/sudharma/blob/main/SECURITY.md">
          <h2>Security</h2>
          <p>Use the project security guidance for vulnerability reporting.</p>
          <span className="text-link">Read SECURITY.md ↗</span>
        </a>
        <a className="info-card linked-card" href="https://github.com/sudharma-networks/sudharma/blob/main/LICENSE">
          <h2>License</h2>
          <p>Read the Apache-2.0 license terms directly in the repository.</p>
          <span className="text-link">Read LICENSE ↗</span>
        </a>
      </section>
    </div>
  );
}
