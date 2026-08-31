import { test, expect } from "@playwright/test";
import { PUBLIC_FAUCET_API_BASE_URL } from "../lib/faucet-config";

const faucetBase = PUBLIC_FAUCET_API_BASE_URL;
const address = "a".repeat(40);
const transactionId = "f".repeat(64);

async function stubFaucet(page: import("@playwright/test").Page, options: { enabled?: boolean; ready?: boolean } = {}) {
  await page.route(`${faucetBase}/v1/faucet/info`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        enabled: options.enabled ?? true,
        challenge_address: address,
        initial_grant_sudh: 100,
        challenge_send_sudh: 25,
        challenge_reward_sudh: 50,
        max_rounds: 5,
        cooldown_hours: 24,
        testnet_only: true
      })
    });
  });
  await page.route(`${faucetBase}/v1/faucet/health`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ ready: options.ready ?? true })
    });
  });
}

test("Faucet page requests a testnet grant and links to the explorer", async ({ page }) => {
  await stubFaucet(page);
  await page.route(`${faucetBase}/v1/faucet/request`, async (route) => {
    expect(route.request().method()).toBe("POST");
    expect(route.request().postData()).toBe(JSON.stringify({ address }));
    await route.fulfill({
      status: 202,
      contentType: "application/json",
      body: JSON.stringify({ address, amount_sudh: 100, transaction_id: transactionId, status: "submitted" })
    });
  });

  await page.goto("/faucet");
  await expect(page.getByRole("heading", { name: /request testnet sudh safely/i })).toBeVisible();
  await expect(page.getByText(/faucet enabled and ready/i)).toBeVisible();

  await page.getByLabel(/sudharma address/i).fill(address);
  await page.getByRole("button", { name: /request 100 test sudh/i }).click();

  await expect(page.getByText(/100 test sudh submitted/i)).toBeVisible();
  await expect(page.getByRole("link", { name: new RegExp(transactionId.slice(0, 12)) }))
    .toHaveAttribute("href", `/explorer/tx?id=${transactionId}`);
});

test("Faucet page rejects malformed addresses before calling the API", async ({ page }) => {
  await stubFaucet(page);
  let requestCalls = 0;
  await page.route(`${faucetBase}/v1/faucet/request`, async (route) => {
    requestCalls += 1;
    await route.fulfill({ status: 202, contentType: "application/json", body: "{}" });
  });

  await page.goto("/faucet");
  await expect(page.getByText(/faucet enabled and ready/i)).toBeVisible();
  await page.getByLabel(/sudharma address/i).fill("not-an-address");
  await page.getByRole("button", { name: /request 100 test sudh/i }).click();

  await expect(page.locator("p.explorer-error")).toContainText(/valid 40-character lowercase hex/i);
  expect(requestCalls).toBe(0);
});

test("Faucet page reports a disabled faucet without offering a request", async ({ page }) => {
  await stubFaucet(page, { enabled: false, ready: false });
  await page.goto("/faucet");
  await expect(page.getByText(/faucet temporarily disabled/i)).toBeVisible();
  await expect(page.getByRole("button", { name: /request 100 test sudh/i })).toBeDisabled();
});
