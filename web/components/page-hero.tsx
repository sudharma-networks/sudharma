import { ReadinessBadge } from "@/components/readiness-badge";

export function PageHero({ eyebrow, title, description, children }: { eyebrow: string; title: string; description: string; children?: React.ReactNode }) {
  return (
    <section className="page-hero">
      <ReadinessBadge />
      <p className="eyebrow">{eyebrow}</p>
      <h1>{title}</h1>
      <p className="lead">{description}</p>
      {children ? <div className="hero-actions">{children}</div> : null}
    </section>
  );
}
