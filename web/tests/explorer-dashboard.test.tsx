import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { ExplorerDashboard } from "@/components/explorer-dashboard";

const apiBase = "https://explorer-api.example.test";

function jsonResponse(body: unknown, status = 200) {
  return Promise.resolve(new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" }
  }));
}

afterEach(() => {
  vi.unstubAllGlobals();
});

function mockExplorerFetch() {
  return vi.fn((input: RequestInfo | URL) => {
    const url = String(input);
    if (url === `${apiBase}/v1/explorer/status`) {
      return jsonResponse({
        network: "sudharma",
        coin: "Sudharma",
        symbol: "SUDH",
        height: 42,
        tip_hash: "a".repeat(64),
        total_work: "43",
        peers: 2,
        mempool: 3,
        issued_supply: 125_000_000,
        data_sources: ["seed-1", "seed-2", "mempool", "demand-miner"],
      });
    }
    if (url === `${apiBase}/v1/explorer/blocks?limit=8`) {
      return jsonResponse({
        blocks: [{
          height: 42,
          hash: "a".repeat(64),
          timestamp: 1_788_000_000,
          previous_hash: "b".repeat(64),
          merkle_root: "c".repeat(64),
          difficulty: 1,
          nonce: 7,
          miner_address: "d".repeat(40),
          transaction_count: 1
        }]
      });
    }
    if (url === `${apiBase}/v1/explorer/transactions?limit=8`) {
      return jsonResponse({
        transactions: [{
          transaction: {
            id: "e".repeat(64),
            from: "1".repeat(40),
            to: "2".repeat(40),
            amount: 50_000_000,
            fee: 50_000,
            nonce: 1
          },
          status: "confirmed",
          block_height: 42,
          block_hash: "a".repeat(64),
          block_timestamp: 1_788_000_000,
          confirmations: 1
        }]
      });
    }
    if (url === `${apiBase}/v1/explorer/mempool?limit=8`) {
      return jsonResponse({
        count: 1,
        transactions: [{
          transaction: {
            id: "f".repeat(64),
            from: "3".repeat(40),
            to: "4".repeat(40),
            amount: 10_000_000,
            fee: 10_000,
            nonce: 2
          },
          status: "pending",
          confirmations: 0
        }]
      });
    }
    throw new Error(`unexpected fetch: ${url}`);
  });
}

test("loads live explorer summary, blocks, transactions, and mempool", async () => {
  vi.stubGlobal("fetch", mockExplorerFetch());

  render(<ExplorerDashboard apiBaseUrl={apiBase} pollIntervalMs={60_000} />);

  expect(await screen.findByText("42")).toBeInTheDocument();
  expect(screen.getByText("2", { selector: "strong" })).toBeInTheDocument();
  expect(screen.getByText("3", { selector: "strong" })).toBeInTheDocument();
  expect(screen.getByText("seed-1")).toBeInTheDocument();
  expect(screen.getByRole("link", { name: /block #42/i })).toHaveAttribute("href", `/explorer/block?id=${"a".repeat(64)}`);
  expect(screen.getByRole("link", { name: /eeeeeeeeeeee/i })).toHaveAttribute("href", `/explorer/tx?id=${"e".repeat(64)}`);
  expect(screen.getByRole("link", { name: /ffffffffffff/i })).toHaveAttribute("href", `/explorer/tx?id=${"f".repeat(64)}`);
  expect(screen.getByText("51,000,000,000 SUDH")).toBeInTheDocument();
});

test("searches the live explorer API and surfaces the resolved detail link", async () => {
  const fetchMock = mockExplorerFetch();
  fetchMock.mockImplementation((input: RequestInfo | URL) => {
    const url = String(input);
    if (url === `${apiBase}/v1/explorer/search?q=42`) {
      return jsonResponse({ type: "block", path: "/explorer/block?id=42" });
    }
    return mockExplorerFetch()(input);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<ExplorerDashboard apiBaseUrl={apiBase} pollIntervalMs={60_000} />);
  await screen.findByText("Connected to live testnet data");

  fireEvent.change(screen.getByRole("textbox", { name: /search blockchain/i }), { target: { value: "42" } });
  fireEvent.click(screen.getByRole("button", { name: /search/i }));

  await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
    `${apiBase}/v1/explorer/search?q=42`,
    expect.objectContaining({ cache: "no-store" })
  ));
  expect(await screen.findByRole("link", { name: /open block result/i })).toHaveAttribute("href", "/explorer/block?id=42");
});

test("shows a truthful unavailable state when no public explorer endpoint is configured", () => {
  render(<ExplorerDashboard apiBaseUrl="" pollIntervalMs={60_000} />);
  expect(screen.getByText(/live explorer api is not configured/i)).toBeInTheDocument();
});
