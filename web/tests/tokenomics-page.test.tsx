import { render, screen } from "@testing-library/react";
import Home from "@/app/page";
import SudhPage from "@/app/sudh/page";

it("surfaces the approved mainnet scarcity story on the homepage", () => {
  render(<Home />);
  expect(screen.getByRole("heading", { name: /designed for scarcity/i })).toBeInTheDocument();
  expect(screen.getByText("51M")).toBeInTheDocument();
  expect(screen.getByText(/0 premine/i)).toBeInTheDocument();
  expect(screen.getByRole("link", { name: /explore tokenomics/i })).toHaveAttribute("href", "/sudh");
});

it("explains the approved mainnet tokenomics without presenting it as live testnet economics", () => {
  render(<SudhPage />);
  expect(screen.getByText(/approved mainnet design · implementation in progress\./i)).toBeInTheDocument();
  expect(screen.getByRole("heading", { name: /51 million\. no premine\. predictable supply\./i })).toBeInTheDocument();
  expect(screen.getByText(/10-year mining era/i)).toBeInTheDocument();
  expect(screen.getByText(/0\.09%.*miners/i)).toBeInTheDocument();
  expect(screen.getByText(/0\.01%.*development treasury/i)).toBeInTheDocument();
  expect(screen.getByText(/implementation in progress/i)).toBeInTheDocument();
});
