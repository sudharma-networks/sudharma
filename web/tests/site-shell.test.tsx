import { render, screen } from "@testing-library/react";
import { SiteHeader } from "@/components/site-header";
it("shows the Downloads route and pre-mainnet status", () => { render(<SiteHeader />); expect(screen.getAllByRole("link", { name: "Downloads" })[0]).toHaveAttribute("href", "/downloads"); expect(screen.getByText(/pre-mainnet/i)).toBeInTheDocument(); });
