import Link from "next/link";

export function ReportProblemLink({ component, context }: { component: string; context: string }) {
  const href = `/support?component=${encodeURIComponent(component)}&context=${encodeURIComponent(context)}`;
  return <Link className="report-link" href={href}>Report problem ↗</Link>;
}
