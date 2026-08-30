import { test, expect } from "@playwright/test";
test("homepage has one primary h1 and keyboard-visible navigation", async ({ page }) => { await page.goto("/"); await expect(page.locator("h1")).toHaveCount(1); await page.keyboard.press("Tab"); await expect(page.locator(":focus-visible")).toBeVisible(); });
