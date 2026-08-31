import { render, screen } from "@testing-library/react";
import ExplorerPage from "@/app/explorer/page";
import FaucetPage from "@/app/faucet/page";
import { vi } from "vitest";

vi.stubGlobal("fetch", vi.fn(async () => new Response("{}", { status: 503 })));

it("does not fabricate explorer metrics", () => {
  render(<ExplorerPage />);
  expect(screen.getByText(/Follow Sudharma testnet in real time/i)).toBeInTheDocument();
  expect(screen.queryByText(/live blocks: 12345/i)).not.toBeInTheDocument();
});

it("does not fabricate faucet grant totals", () => {
  render(<FaucetPage />);
  expect(screen.getByText(/Request testnet SUDH safely/i)).toBeInTheDocument();
  expect(screen.queryByText(/1,000,000 Test SUDH/i)).not.toBeInTheDocument();
});
