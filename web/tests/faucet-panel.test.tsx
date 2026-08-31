import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { FaucetPanel } from "@/components/faucet-panel";
import FaucetPage from "@/app/faucet/page";

afterEach(() => {
  vi.restoreAllMocks();
});

const address = "a".repeat(40);

function mockFaucetFetch(overrides: { enabled?: boolean; ready?: boolean; grantError?: string } = {}) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    if (url.endsWith("/v1/faucet/info")) {
      return new Response(JSON.stringify({
        enabled: overrides.enabled ?? true,
        challenge_address: address,
        initial_grant_sudh: 100,
        challenge_send_sudh: 25,
        challenge_reward_sudh: 50,
        max_rounds: 5,
        cooldown_hours: 24,
        testnet_only: true,
      }), { status: 200, headers: { "content-type": "application/json" } });
    }
    if (url.endsWith("/v1/faucet/health")) {
      return new Response(JSON.stringify({ ready: overrides.ready ?? true }), {
        status: 200,
        headers: { "content-type": "application/json" },
      });
    }
    if (url.endsWith("/v1/faucet/request") && init?.method === "POST") {
      if (overrides.grantError) {
        return new Response(JSON.stringify({ error: overrides.grantError }), {
          status: 409,
          headers: { "content-type": "application/json" },
        });
      }
      return new Response(JSON.stringify({
        address,
        amount_sudh: 100,
        transaction_id: "f".repeat(64),
        status: "submitted",
      }), { status: 202, headers: { "content-type": "application/json" } });
    }
    throw new Error(`unexpected fetch ${url}`);
  });
}

test("faucet page shows testnet status and panel", async () => {
  vi.stubGlobal("fetch", mockFaucetFetch());
  render(<FaucetPage />);
  expect(screen.getByText("Testnet")).toBeInTheDocument();
  await waitFor(() => expect(screen.getByText(/Faucet enabled and ready/i)).toBeInTheDocument());
  expect(screen.getByLabelText(/Sudharma address/i)).toBeInTheDocument();
});

test("submits an initial grant and links to the explorer", async () => {
  const fetchMock = mockFaucetFetch();
  vi.stubGlobal("fetch", fetchMock);
  render(<FaucetPanel apiBaseUrl="https://faucet.example.test" />);

  await waitFor(() => expect(screen.getByText(/Faucet enabled and ready/i)).toBeInTheDocument());
  fireEvent.change(screen.getByLabelText(/Sudharma address/i), { target: { value: address } });
  fireEvent.click(screen.getByRole("button", { name: /Request 100 Test SUDH/i }));

  await waitFor(() => expect(screen.getByText(/100 Test SUDH submitted/i)).toBeInTheDocument());
  const link = screen.getByRole("link", { name: /ffffff/i });
  expect(link).toHaveAttribute("href", `/explorer/tx?id=${"f".repeat(64)}`);
  expect(fetchMock).toHaveBeenCalledWith(
    "https://faucet.example.test/v1/faucet/request",
    expect.objectContaining({ method: "POST" }),
  );
});

test("shows API rejection messages", async () => {
  vi.stubGlobal("fetch", mockFaucetFetch({ grantError: "address already received an initial grant" }));
  render(<FaucetPanel apiBaseUrl="https://faucet.example.test" />);
  await waitFor(() => expect(screen.getByText(/Faucet enabled and ready/i)).toBeInTheDocument());
  fireEvent.change(screen.getByLabelText(/Sudharma address/i), { target: { value: address } });
  fireEvent.click(screen.getByRole("button", { name: /Request 100 Test SUDH/i }));
  await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent(/already received/i));
});
