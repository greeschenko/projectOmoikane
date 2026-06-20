import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test("redirects to /login when not authenticated", async ({ page }) => {
  await page.goto("/settings");
  await expect(page).toHaveURL(/\/login/);
});

test.describe("Password change", () => {
  test("form has current, new, and confirm password fields", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/settings");
    await expect(page.getByLabel(/^Current Password\b/)).toBeVisible();
    await expect(page.getByLabel(/^New Password\b/)).toBeVisible();
    await expect(page.getByLabel(/^Confirm New Password\b/)).toBeVisible();
  });

  test("shows error for wrong current password", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/settings");
    await page.getByLabel(/^Current Password\b/).fill("WrongPassword123!");
    await page.getByLabel(/^New Password\b/).fill("NewPass123!");
    await page.getByLabel(/^Confirm New Password\b/).fill("NewPass123!");
    await page.getByRole("button", { name: /^change password$/i }).click();
    await expect(page.getByText(/current password.*(incorrect|wrong|invalid|not match)/i)).toBeVisible();
  });

  test("shows error when new passwords do not match", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/settings");
    await page.getByLabel(/^Current Password\b/).fill("SecurePass123!");
    await page.getByLabel(/^New Password\b/).fill("NewPass123!");
    await page.getByLabel(/^Confirm New Password\b/).fill("DifferentPass1!");
    await page.getByRole("button", { name: /^change password$/i }).click();
    await expect(page.getByText(/passwords? do not match|passwords? must match/i)).toBeVisible();
  });

  test("shows success message on valid change", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/settings");
    await page.getByLabel(/^Current Password\b/).fill("SecurePass123!");
    await page.getByLabel(/^New Password\b/).fill("NewPass123!");
    await page.getByLabel(/^Confirm New Password\b/).fill("NewPass123!");
    await page.getByRole("button", { name: /^change password$/i }).click();
    await expect(page.getByText(/password.*(changed|updated|success)/i)).toBeVisible();
    await page.getByLabel(/^Current Password\b/).fill("NewPass123!");
    await page.getByLabel(/^New Password\b/).fill("SecurePass123!");
    await page.getByLabel(/^Confirm New Password\b/).fill("SecurePass123!");
    await page.getByRole("button", { name: /^change password$/i }).click();
    await expect(page.getByText(/password.*(changed|updated|success)/i)).toBeVisible();
  });

  test("can login with new password", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/settings");
    await page.getByLabel(/^Current Password\b/).fill("SecurePass123!");
    await page.getByLabel(/^New Password\b/).fill("NewPass123!");
    await page.getByLabel(/^Confirm New Password\b/).fill("NewPass123!");
    await page.getByRole("button", { name: /^change password$/i }).click();
    await expect(page.getByText(/password.*(changed|updated|success)/i)).toBeVisible();

    await page.context().clearCookies();
    await page.goto("/login", { waitUntil: "networkidle" });

    await page.getByLabel("Email").fill("admin@example.com");
    await page.getByLabel("Password").fill("SecurePass123!");
    await page.getByRole("button", { name: /sign in/i }).click();
    await expect(page.getByText(/invalid|incorrect/i)).toBeVisible();

    await page.getByLabel("Email").fill("admin@example.com");
    await page.getByLabel("Password").fill("NewPass123!");
    await page.getByRole("button", { name: /sign in/i }).click();
    await expect(page).toHaveURL(/\/admin/);
  });
});
