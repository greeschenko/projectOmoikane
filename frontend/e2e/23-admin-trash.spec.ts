import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Admin Trash", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("sidebar has Trash nav link", async ({ page }) => {
    await expect(page.getByRole("link", { name: /trash/i })).toBeVisible();
  });

  test("trash page loads and shows items", async ({ page }) => {
    await page.goto("/admin/trash");
    await expect(page.getByRole("heading", { name: /trash/i })).toBeVisible();
  });

  test("hard delete and restore buttons work", async ({ page }) => {
    // Navigate to trash page
    await page.goto("/admin/trash");
    await expect(page.getByRole("heading", { name: /trash/i })).toBeVisible();

    // Check we have Restore and Delete Forever buttons in the table
    const restoreBtns = page.locator("table tbody tr").first().getByRole("button", { name: /restore/i });
    const deleteBtns = page.locator("table tbody tr").first().getByRole("button", { name: /delete forever/i });
    await expect(restoreBtns).toBeVisible();
    await expect(deleteBtns).toBeVisible();
  });
});
