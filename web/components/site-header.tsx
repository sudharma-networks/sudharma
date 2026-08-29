import Link from "next/link";
import { PRIMARY_NAV } from "@/lib/navigation";
import { PROJECT_NAME } from "@/lib/project";
import { ReadinessBadge } from "@/components/readiness-badge";

export function SiteHeader() {
  return (
    <header className="site-header">
      <div className="header-inner">
        <Link className="brand" href="/" aria-label="Sudharma Network home">
          <img src="/sudharma-logo.png" alt="" width="40" height="40" />
          <span>{PROJECT_NAME}</span>
        </Link>
        <div className="desktop-status"><ReadinessBadge /></div>
        <details className="nav-disclosure">
          <summary aria-label="Open site navigation"><span>Menu</span><span className="menu-lines" aria-hidden="true">☰</span></summary>
          <nav aria-label="Primary navigation">
            {PRIMARY_NAV.map(([label, href]) => <Link key={href} href={href}>{label}</Link>)}
          </nav>
        </details>
      </div>
      <nav className="desktop-nav" aria-label="Primary navigation">
        {PRIMARY_NAV.map(([label, href]) => <Link key={href} href={href}>{label}</Link>)}
      </nav>
    </header>
  );
}
