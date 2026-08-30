import { render, screen } from "@testing-library/react";
import { test, vi } from "vitest";
import ExplorerPage from "@/app/explorer/page";

function jsonResponse(body: unknown) {
  return Promise.resolve(new Response(JSON.stringify(body), {
    status: 200,
    headers: { "content-type": "application/json" }
  }));
}

test("explorer page presents live testnet framing without fabricated metrics", async () => {
  vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL) => {
    const url = String(input);
    if (url.endsWith("/v1/explorer/status")) {
      return jsonResponse({ network: "sudharma", coin: "Sudharma", symbol: "SUDH", height: 12, tip_hash: "a".repeat(64), total_work: "13", peers: 1, mempool: 0, issued_supply: 60_000_000_000 });
    }
    if (url.includes("/v1/explorer/blocks")) return jsonResponse({ blocks: [] });
    if (url.includes("/v1/explorer/transactions")) return jsonResponse({ transactions: [] });
    throw new Error(`unexpected fetch: ${url}`);
  }));

  render(<ExplorerPage />);
  expect(await screen.findByText(/follow sudharma testnet in real time/i)).toBeInTheDocument();
  expect(screen.queryByText(/live blocks: 12345/i)).not.toBeInTheDocument();
});
