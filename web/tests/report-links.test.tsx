import { render, screen } from "@testing-library/react";
import { ReportProblemLink } from "@/components/report-problem-link";
it("encodes safe contextual support links", () => { render(<ReportProblemLink component="Mining" context="NVIDIA" />); expect(screen.getByRole("link", { name: /report problem/i })).toHaveAttribute("href", "/support?component=Mining&context=NVIDIA"); });
