import { fireEvent, render, screen } from "@testing-library/react";
import { SiteHeader } from "@/components/site-header";

describe("SiteHeader mobile navigation", () => {
  it("exposes tokenomics as a first-class navigation destination", () => {
    render(<SiteHeader />);

    expect(screen.getAllByRole("link", { name: "Tokenomics" }).length).toBeGreaterThan(0);
    expect(screen.getAllByRole("link", { name: "Tokenomics" })[0]).toHaveAttribute("href", "/sudh");
  });

  it("closes the open mobile menu when a navigation link is selected", () => {
    render(<SiteHeader />);

    const summary = screen.getByLabelText("Open site navigation");
    const disclosure = summary.closest("details");
    expect(disclosure).not.toBeNull();

    disclosure!.open = true;
    expect(disclosure).toHaveAttribute("open");

    const mobileNav = disclosure!.querySelector('nav[aria-label="Primary navigation"]');
    expect(mobileNav).not.toBeNull();

    const firstLink = mobileNav!.querySelector("a");
    expect(firstLink).not.toBeNull();

    fireEvent.click(firstLink!);

    expect(disclosure).not.toHaveAttribute("open");
  });
});
