import { render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import ExplorerPage from "@/app/explorer/page";
import { PUBLIC_EXPLORER_API_BASE_URL } from "@/lib/explorer-config";

function jsonResponse(body: unknown) {
  return Promise.resolve(new Response(JSON.stringify(body), {
    status: 200,
    headers: { "content-type": "application/json" }
  }));
}

const originalExplorerAPI = process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL;

afterEach(() => {
  if (originalExplorerAPI === undefined) {
    delete process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL;
  } else {
    process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL = originalExplorerAPI;
  }
  vi.unstubAllGlobals();
});

test("static explorer page uses the reviewed public bridge when Amplify has no override", async () => {
  delete process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL;
  const fetchMock = vi.fn((input: RequestInfo | URL) => {
    const url = String(input);
    if (url === `${PUBLIC_EXPLORER_API_BASE_URL}/v1/explorer/status`) {
      return jsonResponse({ network: "sudharma", coin: "Sudharma", symbol: "SUDH", height: 12, tip_hash: "a".repeat(64), total_work: "13", peers: 1, mempool: 2, issued_supply: 60_000_000_000 });
    }
    if (url === `${PUBLIC_EXPLORER_API_BASE_URL}/v1/explorer/blocks?limit=8`) return jsonResponse({ blocks: [] });
    if (url === `${PUBLIC_EXPLORER_API_BASE_URL}/v1/explorer/transactions?limit=8`) return jsonResponse({ transactions: [] });
    throw new Error(`unexpected fetch: ${url}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<ExplorerPage />);

  expect(await screen.findByText("Connected to live testnet data")).toBeInTheDocument();
  expect(fetchMock).toHaveBeenCalledWith(`${PUBLIC_EXPLORER_API_BASE_URL}/v1/explorer/status`, expect.objectContaining({ cache: "no-store" }));
});
