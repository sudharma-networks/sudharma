type Status = "Available" | "Testnet" | "Experimental" | "In Development" | "Planned" | "Completed";

export function StatusChip({ status }: { status: Status }) {
  return <span className={`status-chip status-${status.toLowerCase().replaceAll(" ", "-")}`}>{status}</span>;
}
