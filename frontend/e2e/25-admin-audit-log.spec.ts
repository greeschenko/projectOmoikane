import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Admin Audit Log", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("sidebar has Audit Log nav link", async ({ page }) => {
    await expect(page.getByRole("link", { name: /audit log/i })).toBeVisible();
  });

  test("audit log page loads and shows heading", async ({ page }) => {
    await page.goto("/admin/audit-logs");
    await expect(page.getByRole("heading", { name: /audit log/i })).toBeVisible();
  });

  test("audit log page shows table with entries count", async ({ page }) => {
    await page.goto("/admin/audit-logs");
    await expect(page.getByText(/entries/)).toBeVisible();
  });

  test("audit log page has entity filter tabs", async ({ page }) => {
    await page.goto("/admin/audit-logs");
    await expect(page.getByRole("tab", { name: /all/i })).toBeVisible();
    await expect(page.getByRole("tab", { name: /user/i })).toBeVisible();
    await expect(page.getByRole("tab", { name: /page/i })).toBeVisible();
  });

  test("audit log page has search input", async ({ page }) => {
    await page.goto("/admin/audit-logs");
    await expect(page.getByPlaceholder(/search/i)).toBeVisible();
  });
});
