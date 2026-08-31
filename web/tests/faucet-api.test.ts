import { afterEach, expect, test, vi } from "vitest";
import { PUBLIC_FAUCET_API_BASE_URL, resolveFaucetAPIBaseURL } from "@/lib/faucet-config";
import {
  fetchFaucetInfo,
  isValidSudharmaAddress,
  requestFaucetInitialGrant,
} from "@/lib/faucet-api";

afterEach(() => {
  vi.restoreAllMocks();
});

test("defaults to the public faucet bridge URL", () => {
  expect(resolveFaucetAPIBaseURL(undefined)).toBe(PUBLIC_FAUCET_API_BASE_URL);
  expect(resolveFaucetAPIBaseURL(" https://override.example.test/ ")).toBe("https://override.example.test/");
});

test("validates Sudharma addresses", () => {
  expect(isValidSudharmaAddress("a".repeat(40))).toBe(true);
  expect(isValidSudharmaAddress("A".repeat(40))).toBe(false);
  expect(isValidSudharmaAddress("short")).toBe(false);
});

test("fetches faucet info from the public path", async () => {
  const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify({
      enabled: true,
      challenge_address: "a".repeat(40),
      initial_grant_sudh: 100,
      challenge_send_sudh: 25,
      challenge_reward_sudh: 50,
      max_rounds: 5,
      cooldown_hours: 24,
      testnet_only: true,
    }), { status: 200, headers: { "content-type": "application/json" } }),
  );

  const info = await fetchFaucetInfo("https://faucet.example.test");
  expect(info.enabled).toBe(true);
  expect(info.initial_grant_sudh).toBe(100);
  expect(fetchMock).toHaveBeenCalledWith("https://faucet.example.test/v1/faucet/info", { cache: "no-store" });
});

test("posts initial grants with a CORS-safe text/plain body", async () => {
  const address = "b".repeat(40);
  const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify({
      address,
      amount_sudh: 100,
      transaction_id: "c".repeat(64),
      status: "submitted",
    }), { status: 202, headers: { "content-type": "application/json" } }),
  );

  const grant = await requestFaucetInitialGrant("https://faucet.example.test/", address);
  expect(grant.amount_sudh).toBe(100);
  expect(fetchMock).toHaveBeenCalledWith("https://faucet.example.test/v1/faucet/request", {
    method: "POST",
    headers: { "Content-Type": "text/plain;charset=UTF-8" },
    body: JSON.stringify({ address }),
    cache: "no-store",
  });
});

test("surfaces faucet API errors", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify({ error: "already claimed" }), {
      status: 409,
      headers: { "content-type": "application/json" },
    }),
  );

  await expect(requestFaucetInitialGrant("https://faucet.example.test", "d".repeat(40)))
    .rejects.toMatchObject({ name: "FaucetAPIError", status: 409, message: "already claimed" });
});
