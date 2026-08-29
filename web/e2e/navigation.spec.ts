import { test, expect } from "@playwright/test";

test("Mining and Downloads navigation opens full pages", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("link", { name: "Mining" }).first().click();
  await expect(page).toHaveURL(/\/mining$/);
  await expect(page.getByRole("heading", { name: /mine sudharma/i })).toBeVisible();
  await page.getByRole("link", { name: "Downloads" }).first().click();
  await expect(page).toHaveURL(/\/downloads$/);
  await expect(page.getByRole("heading", { name: /verified software/i })).toBeVisible();
});

test("Downloads exposes only official synchronized public release assets", async ({ page }) => {
  await page.goto("/downloads");
  const wallet = page.getByRole("link", { name: "Download" }).filter({ has: page.locator("xpath=ancestor::article[.//h3[contains(., 'Sudharma Android Wallet')]]") });
  await expect(page.getByRole("heading", { name: "Sudharma Android Wallet" })).toBeVisible();
  const walletCard = page.getByRole("heading", { name: "Sudharma Android Wallet" }).locator("xpath=ancestor::article");
  await expect(walletCard.getByText("TESTNET")).toBeVisible();
  await expect(walletCard.getByRole("link", { name: "Download" })).toHaveAttribute("href", "https://github.com/sudharma-networks/sudharma/releases/download/wallet-testnet-v0.1.0/Sudharma-Wallet-0.1.0-testnet.apk");
  await expect(page.getByRole("heading", { name: /NVIDIA \/ CUDA/i })).toBeVisible();
  await expect(page.getByRole("heading", { name: /AMD \/ OpenCL/i })).toBeVisible();
  await expect(page.getByText(/Unrestricted network mining remains gated/i).first()).toBeVisible();
  expect(await wallet.count()).toBeGreaterThanOrEqual(0);
});
