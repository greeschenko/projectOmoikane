import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Admin Messages", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    await page.request.delete("/api/messages");
  });

  test("sidebar has Messages navigation link", async ({ page }) => {
    await page.goto("/admin");
    const nav = page.getByRole("navigation");
    await expect(nav.getByRole("link", { name: /messages/i })).toBeVisible();
  });

  test("page loads at /admin/messages", async ({ page }) => {
    await page.goto("/admin/messages");
    await expect(page.getByRole("heading", { name: /messages/i })).toBeVisible();
  });

  test("has a New Message button", async ({ page }) => {
    await page.goto("/admin/messages");
    await expect(page.getByRole("button", { name: /new message|create message/i })).toBeVisible();
  });

  test("New Message button opens dialog with title and content fields", async ({ page }) => {
    await page.goto("/admin/messages");
    await page.getByRole("button", { name: /new message|create message/i }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await expect(page.getByRole("textbox", { name: /title/i })).toBeVisible();
    await expect(page.getByRole("textbox", { name: /content/i })).toBeVisible();
  });

  test("creating a message shows it in the list", async ({ page }) => {
    await page.goto("/admin/messages");
    await page.getByRole("button", { name: /new message|create message/i }).click();
    await page.getByRole("textbox", { name: /title/i }).fill("Test Broadcast");
    await page.getByRole("textbox", { name: /content/i }).fill("This is a test broadcast message.");
    await page.getByRole("button", { name: /send|create|submit|save/i }).click();
    await expect(page.getByRole("dialog")).not.toBeVisible({ timeout: 10000 });
    await expect(page.getByRole("heading", { name: /test broadcast/i })).toBeVisible();
  });
});
