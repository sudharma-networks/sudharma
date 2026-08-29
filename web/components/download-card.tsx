import type { DownloadArtifact } from "@/lib/downloads";
import { StatusChip } from "@/components/status-chip";
import { ReportProblemLink } from "@/components/report-problem-link";

const statusLabel = { available: "Available", "in-development": "In Development", planned: "Planned" } as const;

export function DownloadCard({ artifact }: { artifact: DownloadArtifact }) {
  return (
    <article className="download-card">
      <div className="card-top"><span className="mono">{artifact.kind.toUpperCase()}</span><StatusChip status={statusLabel[artifact.status]} /></div>
      <h3>{artifact.name}</h3>
      <dl><div><dt>Version</dt><dd>{artifact.version}</dd></div><div><dt>Platform</dt><dd>{artifact.platform}</dd></div><div><dt>Architecture</dt><dd>{artifact.architecture}</dd></div></dl>
      {artifact.sha256 ? <p className="checksum">SHA256 {artifact.sha256}</p> : null}
      <div className="card-actions">
        {artifact.status === "available" && artifact.downloadUrl ? <a className="button small" href={artifact.downloadUrl}>Download</a> : <span className="muted">No public binary published yet.</span>}
        {artifact.sourceUrl ? <a className="text-link" href={artifact.sourceUrl}>Source ↗</a> : null}
      </div>
      <ReportProblemLink component="Downloads" context={`${artifact.id}:${artifact.version}`} />
    </article>
  );
}
