import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import ExplorerBlockPage from "@/app/explorer/block/page";
import ExplorerTransactionPage from "@/app/explorer/tx/page";
import ExplorerAddressPage from "@/app/explorer/address/page";
import { PUBLIC_EXPLORER_API_BASE_URL } from "@/lib/explorer-config";

const blockHash = "a".repeat(64);
const txId = "b".repeat(64);
const address = "c".repeat(40);

function jsonResponse(body: unknown) {
  return Promise.resolve(new Response(JSON.stringify(body), {
    status: 200,
    headers: { "content-type": "application/json" }
  }));
}

const originalExplorerAPI = process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL;

afterEach(() => {
  cleanup();
  window.history.replaceState({}, "", "/");
  if (originalExplorerAPI === undefined) delete process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL;
  else process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL = originalExplorerAPI;
  vi.unstubAllGlobals();
});

test("block detail route uses the reviewed public bridge without an Amplify override", async () => {
  delete process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL;
  window.history.replaceState({}, "", `/explorer/block?id=${blockHash}`);
  const fetchMock = vi.fn((input: RequestInfo | URL) => {
    expect(String(input)).toBe(`${PUBLIC_EXPLORER_API_BASE_URL}/v1/explorer/blocks/${blockHash}`);
    return jsonResponse({
      height: 12,
      hash: blockHash,
      timestamp: 1_787_850_935,
      previous_hash: "d".repeat(64),
      merkle_root: "e".repeat(64),
      difficulty: 1,
      nonce: 0,
      miner_address: address,
      transaction_count: 0,
      transactions: []
    });
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<ExplorerBlockPage />);
  expect(await screen.findByRole("heading", { name: "Block #12" })).toBeInTheDocument();
});

test("transaction detail route uses the reviewed public bridge without an Amplify override", async () => {
  delete process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL;
  window.history.replaceState({}, "", `/explorer/tx?id=${txId}`);
  const fetchMock = vi.fn((input: RequestInfo | URL) => {
    expect(String(input)).toBe(`${PUBLIC_EXPLORER_API_BASE_URL}/v1/explorer/transactions/${txId}`);
    return jsonResponse({
      transaction: { id: txId, from: address, to: "f".repeat(40), amount: 100_000_000, fee: 100_000, nonce: 1 },
      status: "confirmed",
      block_height: 12,
      block_hash: blockHash,
      block_timestamp: 1_787_850_935,
      confirmations: 1
    });
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<ExplorerTransactionPage />);
  expect(await screen.findByRole("heading", { name: "Transaction detail" })).toBeInTheDocument();
  expect(screen.getByText("Confirmed", { exact: true })).toBeInTheDocument();
});

test("address detail route uses the reviewed public bridge without an Amplify override", async () => {
  delete process.env.NEXT_PUBLIC_EXPLORER_API_BASE_URL;
  window.history.replaceState({}, "", `/explorer/address?address=${address}`);
  const fetchMock = vi.fn((input: RequestInfo | URL) => {
    expect(String(input)).toBe(`${PUBLIC_EXPLORER_API_BASE_URL}/v1/explorer/addresses/${address}?limit=20`);
    return jsonResponse({ address, balance: 0, confirmed_nonce: 0, next_nonce: 1, transactions: [] });
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<ExplorerAddressPage />);
  expect(await screen.findByRole("heading", { name: "Account activity" })).toBeInTheDocument();
});
