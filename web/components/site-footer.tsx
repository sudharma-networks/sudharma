import Link from "next/link";
import { FOOTER_NAV, PRIMARY_NAV } from "@/lib/navigation";
import { PROJECT_NAME } from "@/lib/project";
import { ReadinessBadge } from "@/components/readiness-badge";

export function SiteFooter() {
  return (
    <footer className="site-footer">
      <div className="footer-grid">
        <div>
          <Link className="brand" href="/"><img src="/sudharma-logo.png" alt="" width="42" height="42" /><span>{PROJECT_NAME}</span></Link>
          <p className="footer-copy">Open-source Proof-of-Work infrastructure built in public.</p>
          <ReadinessBadge />
        </div>
        <div><h2>Explore</h2>{PRIMARY_NAV.slice(1, 8).map(([label, href]) => <Link key={href} href={href}>{label}</Link>)}</div>
        <div><h2>Project</h2>{PRIMARY_NAV.slice(8).map(([label, href]) => <Link key={href} href={href}>{label}</Link>)}</div>
        <div><h2>Trust</h2>{FOOTER_NAV.map(([label, href]) => <a key={href} href={href} target="_blank" rel="noreferrer">{label}</a>)}</div>
      </div>
      <div className="footer-bottom">© 2026 Sudharma Network · Pre-mainnet software. Verify before use.</div>
    </footer>
  );
}
