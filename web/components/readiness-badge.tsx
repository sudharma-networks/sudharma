import { PROJECT_STATUS } from "@/lib/project";

export function ReadinessBadge() {
  return <span className="readiness-badge"><span className="status-dot" aria-hidden="true" />{PROJECT_STATUS}</span>;
}
