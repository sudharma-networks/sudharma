import { existsSync } from "node:fs";
import { resolve } from "node:path";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import {
  ExplorerAddressDetail,
  ExplorerBlockDetail,
  ExplorerTransactionDetail
} from "@/components/explorer-details";

const apiBase = "https://explorer-api.example.test";
const blockHash = "a".repeat(64);
const previousHash = "b".repeat(64);
const merkleRoot = "c".repeat(64);
const txID = "d".repeat(64);
const address = "1".repeat(40);
const recipient = "2".repeat(40);

function jsonResponse(body: unknown, status = 200) {
  return Promise.resolve(new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" }
  }));
}

function confirmedTransaction(id = txID) {
  return {
    transaction: {
      id,
      from: address,
      to: recipient,
      amount: 150_000_000,
      fee: 25_000,
      nonce: 7
    },
    status: "confirmed",
    block_height: 42,
    block_hash: blockHash,
    block_timestamp: 1_788_000_000,
    confirmations: 6
  };
}

afterEach(() => {
  vi.unstubAllGlobals();
});

test("ships static-export-compatible block, transaction and address routes", () => {
  expect(existsSync(resolve(process.cwd(), "app/explorer/block/page.tsx"))).toBe(true);
  expect(existsSync(resolve(process.cwd(), "app/explorer/tx/page.tsx"))).toBe(true);
  expect(existsSync(resolve(process.cwd(), "app/explorer/address/page.tsx"))).toBe(true);
});

test("loads canonical block details and links its transactions", async () => {
  const fetchMock = vi.fn((input: RequestInfo | URL) => {
    expect(String(input)).toBe(`${apiBase}/v1/explorer/blocks/${blockHash}`);
    return jsonResponse({
      height: 42,
      hash: blockHash,
      timestamp: 1_788_000_000,
      previous_hash: previousHash,
      merkle_root: merkleRoot,
      difficulty: 9,
      nonce: 77,
      miner_address: address,
      transaction_count: 1,
      transactions: [confirmedTransaction().transaction]
    });
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<ExplorerBlockDetail apiBaseUrl={apiBase} blockId={blockHash} />);
  expect(screen.getByText(/loading block details/i)).toBeInTheDocument();

  expect(await screen.findByRole("heading", { name: /block #42/i })).toBeInTheDocument();
  expect(screen.getByText(blockHash)).toBeInTheDocument();
  expect(screen.getByRole("link", { name: /dddddddddddd/i })).toHaveAttribute("href", `/explorer/tx?id=${txID}`);
  expect(screen.getByRole("link", { name: /111111111111/i })).toHaveAttribute("href", `/explorer/address?address=${address}`);
});

test("shows a canonical not-found state for an unknown block", async () => {
  vi.stubGlobal("fetch", vi.fn(() => jsonResponse({ error: "block not found" }, 404)));
  render(<ExplorerBlockDetail apiBaseUrl={apiBase} blockId={blockHash} />);
  expect(await screen.findByText(/block not found on the canonical chain/i)).toBeInTheDocument();
});

test("shows pending transaction status without inventing block metadata", async () => {
  vi.stubGlobal("fetch", vi.fn(() => jsonResponse({
    transaction: {
      id: txID,
      from: address,
      to: recipient,
      amount: 50_000_000,
      fee: 50_000,
      nonce: 8
    },
    status: "pending",
    confirmations: 0
  })));

  render(<ExplorerTransactionDetail apiBaseUrl={apiBase} transactionId={txID} />);
  expect(await screen.findByText(/waiting in the seed-node mempool/i)).toBeInTheDocument();
  expect(screen.getByText(/pending \(mempool\)/i)).toBeInTheDocument();
  expect(screen.queryByText(/block height/i)).not.toBeInTheDocument();
});

test("shows a truthful error state when transaction detail is unavailable", async () => {
  vi.stubGlobal("fetch", vi.fn(() => jsonResponse({ error: "unavailable" }, 503)));
  render(<ExplorerTransactionDetail apiBaseUrl={apiBase} transactionId={txID} />);
  expect(await screen.findByRole("alert")).toHaveTextContent(/transaction details unavailable/i);
});

test("loads address balance and appends older history using the opaque cursor", async () => {
  const olderID = "e".repeat(64);
  const fetchMock = vi.fn((input: RequestInfo | URL) => {
    const url = String(input);
    if (url === `${apiBase}/v1/explorer/addresses/${address}?limit=20`) {
      return jsonResponse({
        address,
        balance: 325_000_000,
        confirmed_nonce: 7,
        next_nonce: 8,
        transactions: [confirmedTransaction()],
        next_cursor: "cursor-1"
      });
    }
    if (url === `${apiBase}/v1/explorer/addresses/${address}?limit=20&cursor=cursor-1`) {
      return jsonResponse({
        address,
        balance: 325_000_000,
        confirmed_nonce: 7,
        next_nonce: 8,
        transactions: [confirmedTransaction(olderID)]
      });
    }
    throw new Error(`unexpected fetch: ${url}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<ExplorerAddressDetail apiBaseUrl={apiBase} address={address} />);
  expect(await screen.findByText("3.25 SUDH")).toBeInTheDocument();
  expect(screen.getByRole("link", { name: /dddddddddddd/i })).toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: /load older transactions/i }));
  expect(await screen.findByRole("link", { name: /eeeeeeeeeeee/i })).toBeInTheDocument();
  expect(fetchMock).toHaveBeenCalledWith(
    `${apiBase}/v1/explorer/addresses/${address}?limit=20&cursor=cursor-1`,
    expect.objectContaining({ cache: "no-store" })
  );
});

test("restarts address history from the canonical tip when a cursor becomes stale", async () => {
  let firstPageCalls = 0;
  const fetchMock = vi.fn((input: RequestInfo | URL) => {
    const url = String(input);
    if (url === `${apiBase}/v1/explorer/addresses/${address}?limit=20`) {
      firstPageCalls += 1;
      return jsonResponse({
        address,
        balance: 100_000_000,
        confirmed_nonce: 1,
        next_nonce: 2,
        transactions: [confirmedTransaction(firstPageCalls === 1 ? txID : "f".repeat(64))],
        next_cursor: firstPageCalls === 1 ? "stale-cursor" : ""
      });
    }
    if (url.endsWith("cursor=stale-cursor")) {
      return jsonResponse({ error: "explorer cursor is stale; restart pagination" }, 409);
    }
    throw new Error(`unexpected fetch: ${url}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<ExplorerAddressDetail apiBaseUrl={apiBase} address={address} />);
  await screen.findByRole("link", { name: /dddddddddddd/i });
  fireEvent.click(screen.getByRole("button", { name: /load older transactions/i }));

  expect(await screen.findByText(/chain changed while paging/i)).toBeInTheDocument();
  expect(await screen.findByRole("link", { name: /ffffffffffff/i })).toBeInTheDocument();
  await waitFor(() => expect(firstPageCalls).toBe(2));
});
