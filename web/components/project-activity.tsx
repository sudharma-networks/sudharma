import status from "../public/data/project-status.json";

export function ProjectActivity() {
  const shortSha = status.latestCommitSha ? status.latestCommitSha.slice(0, 8) : "Unavailable";
  return (
    <section className="section-shell section-block">
      <div className="section-heading">
        <p className="eyebrow">PUBLIC GITHUB ACTIVITY</p>
        <h2>Repository activity, synchronized automatically.</h2>
        <p>This is informational pre-mainnet activity only. It does not mean a feature is production-ready.</p>
      </div>
      <div className="stats-grid">
        <div className="stat"><span>Latest public release</span><strong>{status.latestReleaseTag ?? "No release"}</strong></div>
        <div className="stat"><span>Latest main commit</span><strong>{shortSha}</strong></div>
        <div className="stat"><span>Source updated</span><strong>{status.generatedAt ? status.generatedAt.slice(0, 10) : "Unknown"}</strong></div>
      </div>
      <div className="hero-actions">
        {status.latestReleaseUrl ? <a className="text-link" href={status.latestReleaseUrl}>View latest release ↗</a> : null}
        {status.latestCommitUrl ? <a className="text-link" href={status.latestCommitUrl}>View latest commit ↗</a> : null}
      </div>
    </section>
  );
}
