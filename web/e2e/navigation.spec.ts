import { test, expect } from "@playwright/test";
import releaseSnapshot from "../public/data/github-releases.json";

const visitorEndpoint = "https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com/v1/website/visitors";

const walletArtifact = releaseSnapshot.artifacts.find(
  (artifact) => artifact.slot === "android-wallet" && artifact.status === "available",
);

test("Mining and Downloads navigation opens full pages", async ({ page }) => {
  await page.route(visitorEndpoint, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ total: 123 })
    });
  });
  await page.goto("/");
  await expect(page.getByText("Website Visitors")).toBeVisible();
  await expect(page.getByText("123", { exact: true })).toBeVisible();
  await page.getByRole("link", { name: "Mining" }).first().click();
  await expect(page).toHaveURL(/\/mining$/);
  await expect(page.getByRole("heading", { name: /mine sudharma/i })).toBeVisible();
  await page.getByRole("link", { name: "Downloads" }).first().click();
  await expect(page).toHaveURL(/\/downloads$/);
  await expect(page.getByRole("heading", { name: /verified software/i })).toBeVisible();
});

test("Downloads exposes only official synchronized public release assets", async ({ page }) => {
  expect(walletArtifact?.downloadUrl).toMatch(/^https:\/\/github.com\/sudharma-networks\/sudharma\/releases\/download\//);

  await page.goto("/downloads");
  const walletCard = page.locator("article.download-card").filter({ hasText: walletArtifact?.version ?? "wallet-testnet" }).first();
  await expect(walletCard.getByRole("heading", { name: "Sudharma Android Wallet" })).toBeVisible();
  await expect(walletCard.getByText("TESTNET", { exact: true })).toBeVisible();
  await expect(walletCard.getByRole("link", { name: "Download" })).toHaveAttribute("href", walletArtifact!.downloadUrl!);
  await expect(page.getByRole("heading", { name: /NVIDIA \/ CUDA/i })).toBeVisible();
  await expect(page.getByRole("heading", { name: /AMD \/ OpenCL/i })).toBeVisible();
  await expect(page.getByText(/Unrestricted network mining remains gated/i).first()).toBeVisible();
});
