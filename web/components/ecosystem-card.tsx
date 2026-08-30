import Link from "next/link";

export function EcosystemCard({ number, title, description, href, links }: { number: string; title: string; description: string; href: string; links: string }) {
  return (
    <Link href={href} className="ecosystem-card">
      <span className="card-number">{number}</span>
      <h3>{title}</h3>
      <p>{description}</p>
      <span className="card-links">{links}</span>
      <span className="card-arrow" aria-hidden="true">↗</span>
    </Link>
  );
}
