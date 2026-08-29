import { render, screen } from "@testing-library/react";
import AboutPage from "@/app/about/page";
import { SiteHeader } from "@/components/site-header";

it("links About Us from the main navigation", () => {
  render(<SiteHeader />);
  expect(screen.getAllByRole("link", { name: "About Us" })[0]).toHaveAttribute("href", "/about");
});

it("presents Sudharma as a student-built open project from India", () => {
  render(<AboutPage />);
  expect(screen.getByRole("heading", { name: /built in india/i })).toBeInTheDocument();
  expect(screen.getByText(/students from India/i)).toBeInTheDocument();
  expect(screen.getByRole("heading", { name: /use sudharma/i })).toBeInTheDocument();
  expect(screen.getByRole("heading", { name: /mine sudharma/i })).toBeInTheDocument();
  expect(screen.getByRole("heading", { name: /build on sudharma/i })).toBeInTheDocument();
  expect(screen.getByText(/does not promise guaranteed financial returns/i)).toBeInTheDocument();
  expect(screen.getByRole("link", { name: /explore downloads/i })).toHaveAttribute("href", "/downloads");
  expect(screen.getByRole("link", { name: /build with us/i })).toHaveAttribute("href", "/developers");
});
