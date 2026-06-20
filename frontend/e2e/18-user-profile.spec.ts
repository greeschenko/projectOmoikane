import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("User Profile", () => {
  test("settings page has name and email fields", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/settings");
    await expect(page.getByLabel(/^name\b/i)).toBeVisible();
    await expect(page.getByLabel(/^email\b/i)).toBeVisible();
  });

  test("can update display name", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/settings");
    const nameInput = page.getByLabel(/^name\b/i);
    await nameInput.clear();
    await nameInput.fill("Updated Name");
    await page.getByRole("button", { name: /save profile|update profile|save changes/i }).click();
    await expect(page.getByText(/saved|updated|success/i)).toBeVisible();

    await page.reload();
    await expect(page.getByLabel(/^name\b/i)).toHaveValue("Updated Name");
  });

  test("can update email", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/settings");
    const emailInput = page.getByLabel(/^email\b/i);
    await emailInput.clear();
    await emailInput.fill("updated@example.com");
    await page.getByRole("button", { name: /save profile|update profile|save changes/i }).click();
    await expect(page.getByText(/saved|updated|success/i)).toBeVisible();

    // Restore original email so loginAsAdmin still works for other tests
    await emailInput.clear();
    await emailInput.fill("admin@example.com");
    await page.getByRole("button", { name: /save profile|update profile|save changes/i }).click();
    await expect(page.getByText(/saved|updated|success/i)).toBeVisible();
  });

  test("settings page has avatar upload field", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/settings");
    await expect(page.getByRole("button", { name: /avatar|upload avatar|change avatar/i })).toBeVisible();
  });
});
