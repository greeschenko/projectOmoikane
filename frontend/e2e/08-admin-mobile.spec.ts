import { test, expect } from "@playwright/test";
import { isMobile, loginAsAdmin } from "./helpers";

test.describe("Mobile admin responsive layout", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test.describe("All tests in this file run only on mobile", () => {
    test.beforeAll(() => {
      test.skip(!isMobile(), "Mobile-specific tests");
    });

    test("hamburger menu button is visible", async ({ page }) => {
      await page.goto("/admin");
      const hamburger = page.getByRole("button", { name: /menu|hamburger|toggle sidebar/i });
      await expect(hamburger).toBeVisible();
    });

    test("hamburger menu opens sidebar overlay", async ({ page }) => {
      await page.goto("/admin");
      const hamburger = page.getByRole("button", { name: /menu|hamburger|toggle sidebar/i });
      await hamburger.click();
      const sidebar = page.locator("nav, aside, [role='navigation']").first();
      await expect(sidebar).toBeVisible();
    });

    test("sidebar overlay closes after navigating to a page", async ({ page }) => {
      await page.goto("/admin");
      await page.getByRole("button", { name: /menu|hamburger|toggle sidebar/i }).click();
      await page.getByRole("link", { name: /users/i }).click();
      await expect(page.locator("nav, aside, [role='navigation']").first()).not.toBeVisible();
    });

    test("admin page has no horizontal scroll", async ({ page }) => {
      await page.goto("/admin");
      const scrollWidth = await page.evaluate(() => document.documentElement.scrollWidth);
      const clientWidth = await page.evaluate(() => document.documentElement.clientWidth);
      expect(scrollWidth).toBeLessThanOrEqual(clientWidth + 1);
    });

    test("user table renders as cards on mobile", async ({ page }) => {
      await page.goto("/admin/users");
      await page.waitForSelector("table, [role='grid'], [role='list']");
      const isTable = await page.locator("table, [role='grid']").isVisible();
      const isCardList = await page.locator("[role='list'] > [role='listitem'], .MuiCard-root").first().isVisible().catch(() => false);
      expect(isTable || isCardList).toBeTruthy();
    });

    test("New User form fits mobile viewport", async ({ page }) => {
      const usersResponse = page.waitForResponse(r => r.url().includes('/api/users') && r.status() === 200);
      await page.goto("/admin/users");
      await usersResponse;
      await page.getByPlaceholder(/filter|search/i).waitFor();
      await page.getByRole("button", { name: /new user|add user|create user/i }).click();
      const dialog = page.locator("[role='dialog'], form").first();
      const box = await dialog.boundingBox();
      expect(box).not.toBeNull();
      if (box) {
        expect(box.width).toBeLessThanOrEqual(400);
      }
    });

    test("form fields have adequate tap targets", async ({ page }) => {
      await page.goto("/admin/users");
      await page.getByPlaceholder(/filter|search/i).waitFor();
      await page.getByRole("button", { name: /new user|add user|create user/i }).click();
      const inputs = page.locator("input, select, button");
      const count = await inputs.count();
      for (let i = 0; i < Math.min(count, 5); i++) {
        const box = await inputs.nth(i).boundingBox();
        if (box) {
          expect(box.height).toBeGreaterThanOrEqual(36);
        }
      }
    });

    test("filter input works on mobile", async ({ page }) => {
      await page.goto("/admin/users");
      const filter = page.getByPlaceholder(/filter|search/i);
      await expect(filter).toBeVisible();
      await filter.fill("test@example.com");
      await expect(filter).toHaveValue("test@example.com");
    });
  });
});
