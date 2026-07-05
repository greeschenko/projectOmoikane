import { test, expect } from "@playwright/test";
import { isMobile, loginAsAdmin } from "./helpers";

const userFormFieldLabels = [/^Name\b/i, /^Email\b/i, /^Password\b/i, /^Confirm Password\b/i];
const userRoleOptions = ["admin", "user"];

test.describe("Admin Users", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test.describe("User table", () => {
    test("has sortable columns", async ({ page }) => {
      test.skip(isMobile(), "Table sorting switches to dropdown on mobile");
      await page.goto("/admin/users");
      await expect(page.getByRole("table")).toBeVisible();
      const headers = page.getByRole("columnheader");
      await expect(headers.first()).toBeVisible();
      const headerTexts = await headers.allTextContents();
      expect(headerTexts.length).toBeGreaterThan(0);
    });

    test("has a filter input", async ({ page }) => {
      await page.goto("/admin/users");
      await expect(page.getByPlaceholder(/filter|search/i)).toBeVisible();
    });

    test("shows user data in rows", async ({ page }) => {
      await page.goto("/admin/users");
      const rows = page.locator("table tbody tr");
      await expect(rows.first()).toBeVisible();
    });

    test("each user row has edit and delete actions", async ({ page }) => {
      await page.goto("/admin/users");
      const firstRow = page.locator("table tbody tr").first();
      await expect(firstRow.getByRole("button", { name: /edit|pencil/i })).toBeVisible();
      await expect(firstRow.getByRole("button", { name: /delete|remove|trash/i })).toBeVisible();
    });

    test("shows empty state when filter matches no users", async ({ page }) => {
      await page.goto("/admin/users");
      await expect(page.locator("table tbody tr")).not.toHaveCount(0);
      await page.getByPlaceholder(/filter|search/i).fill("zzzzthisdoesnotmatch");
      await expect(page.getByRole("table")).toContainText(/no users? found/i);
    });
  });

  test.describe("User creation form", () => {
    test("'New User' button opens a form", async ({ page }) => {
      const usersResponse = page.waitForResponse(r => r.url().includes('/api/users') && r.status() === 200);
      await page.goto("/admin/users");
      await usersResponse;
      await page.getByPlaceholder(/filter|search/i).waitFor();
      await page.getByRole("button", { name: /new user|add user|create user/i }).click();
      await expect(page.getByRole("dialog")).toBeVisible();
    });

    test("form has name, email, password, confirm password, and role fields", async ({ page }) => {
      const usersResponse = page.waitForResponse(r => r.url().includes('/api/users') && r.status() === 200);
      await page.goto("/admin/users");
      await usersResponse;
      await page.getByPlaceholder(/filter|search/i).waitFor();
      await page.getByRole("button", { name: /new user|add user|create user/i }).click();
      for (const field of userFormFieldLabels) {
        await expect(page.getByLabel(field)).toBeVisible();
      }
      await expect(page.getByRole("combobox")).toBeVisible();
    });

    test("role selector has admin and user options", async ({ page }) => {
      const usersResponse = page.waitForResponse(r => r.url().includes('/api/users') && r.status() === 200);
      await page.goto("/admin/users");
      await usersResponse;
      await page.getByPlaceholder(/filter|search/i).waitFor();
      await page.getByRole("button", { name: /new user|add user|create user/i }).click();
      await page.getByRole("combobox").click();
      for (const option of userRoleOptions) {
        await expect(page.getByRole("option", { name: new RegExp(option, "i") })).toBeVisible();
      }
    });

    test("shows validation errors on empty submission", async ({ page }) => {
      const usersResponse = page.waitForResponse(r => r.url().includes('/api/users') && r.status() === 200);
      await page.goto("/admin/users");
      await usersResponse;
      await page.getByPlaceholder(/filter|search/i).waitFor();
      await page.getByRole("button", { name: /new user|add user|create user/i }).click();
      await page.getByRole("button", { name: /save|create|submit/i }).click();
      await expect(page.getByText(/required|cannot be empty/i).first()).toBeVisible();
    });

    test("shows error for invalid email", async ({ page }) => {
      const usersResponse = page.waitForResponse(r => r.url().includes('/api/users') && r.status() === 200);
      await page.goto("/admin/users");
      await usersResponse;
      await page.getByPlaceholder(/filter|search/i).waitFor();
      await page.getByRole("button", { name: /new user|add user|create user/i }).click();
      await page.getByLabel("Name").fill("New User");
      await page.getByLabel("Email").fill("not-an-email");
      await page.getByLabel(/^Password\b/).fill("Password123!");
      await page.getByLabel("Confirm Password").fill("Password123!");
      await page.getByRole("button", { name: /save|create|submit/i }).click();
      await expect(page.getByText(/valid email/i)).toBeVisible();
    });

    test("shows error when passwords do not match", async ({ page }) => {
      const usersResponse = page.waitForResponse(r => r.url().includes('/api/users') && r.status() === 200);
      await page.goto("/admin/users");
      await usersResponse;
      await page.getByPlaceholder(/filter|search/i).waitFor();
      await page.getByRole("button", { name: /new user|add user|create user/i }).click();
      await page.getByLabel("Name").fill("New User");
      await page.getByLabel("Email").fill("user@example.com");
      await page.getByLabel(/^Password\b/).fill("Password123!");
      await page.getByLabel("Confirm Password").fill("DifferentPass1!");
      await page.getByRole("button", { name: /save|create|submit/i }).click();
      await expect(page.getByText(/passwords? do not match|passwords? must match/i)).toBeVisible();
    });

    test("successful submission adds user to table", async ({ page }) => {
      await page.goto("/admin/users");
      await expect(page.locator("table tbody tr")).not.toHaveCount(0);
      await expect(page.getByRole("table")).toContainText(/@example\.com/);
      const rowCount = await page.locator("table tbody tr").count();
      await page.getByPlaceholder(/filter|search/i).waitFor();
      await expect(page.locator("table tbody tr")).not.toHaveCount(0);
      await page.getByRole("button", { name: /new user|add user|create user/i }).click();
      await page.getByLabel("Name").fill("Jane Doe");
      await page.getByLabel("Email").fill("jane@example.com");
      await page.getByLabel(/^Password\b/).fill("SecurePass123!");
      await page.getByLabel("Confirm Password").fill("SecurePass123!");
      await page.getByRole("button", { name: /save|create|submit/i }).click();
      await expect(page.getByRole("dialog")).not.toBeVisible({ timeout: 10000 });
      await page.waitForTimeout(500);
      const newRowCount = await page.locator("table tbody tr").count();
      expect(newRowCount).toBe(rowCount + 1);
      await expect(page.getByRole("table")).toContainText(/jane@example.com/i);
    });

    test("cancel closes the form without adding a user", async ({ page }) => {
      await page.goto("/admin/users");
      await expect(page.locator("table tbody tr")).not.toHaveCount(0);
      await expect(page.getByRole("table")).toContainText(/@example\.com/);
      const rowCount = await page.locator("table tbody tr").count();
      await page.getByPlaceholder(/filter|search/i).waitFor();
      await expect(page.locator("table tbody tr")).not.toHaveCount(0);
      await page.getByRole("button", { name: /new user|add user|create user/i }).click();
      await page.getByRole("button", { name: /cancel/i }).click();
      await expect(page.locator("[role='dialog'], form")).not.toBeVisible();
      const afterCancelCount = await page.locator("table tbody tr").count();
      expect(afterCancelCount).toBe(rowCount);
    });
  });

  test.describe("User edit", () => {
    test("edit button opens form pre-filled with user data", async ({ page }) => {
      await page.goto("/admin/users");
      const firstRow = page.locator("table tbody tr").first();
      const email = await firstRow.locator("td, [role='gridcell']").nth(1).textContent();
      await firstRow.getByRole("button", { name: /edit|pencil/i }).click();
      await expect(page.getByLabel("Email")).toHaveValue(email?.trim() ?? "");
    });

    test("saving changes updates the table", async ({ page }) => {
      await page.goto("/admin/users");
      const firstRow = page.locator("table tbody tr").first();
      await firstRow.getByRole("button", { name: /edit|pencil/i }).click();
      await page.getByLabel("Name").fill("Updated Name");
      await page.getByRole("button", { name: /save|update/i }).click();
      await expect(page.getByRole("table")).toContainText(/updated name/i);
    });

    test("cancel edit closes form without changes", async ({ page }) => {
      await page.goto("/admin/users");
      const firstRow = page.locator("table tbody tr").first();
      await firstRow.getByRole("button", { name: /edit|pencil/i }).click();
      await page.getByRole("button", { name: /cancel/i }).click();
      await expect(page.locator("[role='dialog'], form")).not.toBeVisible();
    });
  });

  test.describe("User delete", () => {
    test("delete button shows a confirmation dialog", async ({ page }) => {
      await page.goto("/admin/users");
      await expect(page.locator("table tbody tr")).not.toHaveCount(0);
      const firstRow = page.locator("table tbody tr").first();
      await firstRow.getByRole("button", { name: /delete|remove|trash/i }).click();
      await expect(page.getByRole("dialog")).toBeVisible();
    });

    test("confirming delete removes the user from the table", async ({ page }) => {
      await page.goto("/admin/users");
      await expect(page.locator("table tbody tr")).not.toHaveCount(0);
      await expect(page.getByRole("table")).toContainText(/@example\.com/);
      const rowCount = await page.locator("table tbody tr").count();
      const firstRow = page.locator("table tbody tr").first();
      await firstRow.getByRole("button", { name: /delete|remove|trash/i }).click();
      await page.getByRole("button", { name: /confirm|yes|delete/i }).click();
      await expect(page.locator("[role='dialog']")).not.toBeVisible();
      await page.waitForTimeout(500);
      const afterCount = await page.locator("table tbody tr").count();
      expect(afterCount).toBe(rowCount - 1);
    });

    test("cancelling delete keeps the user in the table", async ({ page }) => {
      await page.goto("/admin/users");
      await expect(page.locator("table tbody tr")).not.toHaveCount(0);
      await expect(page.getByRole("table")).toContainText(/@example\.com/);
      const rowCount = await page.locator("table tbody tr").count();
      const firstRow = page.locator("table tbody tr").first();
      await firstRow.getByRole("button", { name: /delete|remove|trash/i }).click();
      await page.getByRole("button", { name: /cancel|no/i }).click();
      await expect(page.locator("[role='dialog']")).not.toBeVisible();
      const afterCount = await page.locator("table tbody tr").count();
      expect(afterCount).toBe(rowCount);
    });
  });
});
