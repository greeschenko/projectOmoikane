import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("User Status", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("user table has a Status column header", async ({ page }) => {
    await page.goto("/admin/users");
    const headers = page.getByRole("columnheader");
    const headerTexts = await headers.allTextContents();
    expect(headerTexts.some((t) => /status/i.test(t))).toBeTruthy();
  });

  test("default status shows Active badge in each row", async ({ page }) => {
    await page.goto("/admin/users");
    const firstRow = page.locator("table tbody tr").first();
    await expect(firstRow).toContainText(/active/i);
  });

  test("edit dialog has Status field and can change to banned", async ({ page }) => {
    // Create a unique user so we don't ban the admin
    const email = `testuser-${Date.now()}@example.com`;
    await page.request.post("/api/users", {
      data: { name: "Test User", email, password: "Password123!", role: "user", status: "active" },
    });
    await page.goto("/admin/users");
    const testRow = page.locator("table tbody tr", { hasText: email });
    await testRow.getByRole("button", { name: /edit|pencil/i }).click();
    const statusSelect = page.getByRole("dialog").getByText("Status").locator("..").getByRole("combobox");
    await expect(statusSelect).toBeVisible();
    await statusSelect.click();
    await page.getByRole("option", { name: /banned/i }).click();
    await page.getByRole("button", { name: /save|update/i }).click();
    await expect(page.getByRole("table")).toContainText(/banned/i);
  });

  test("banned user cannot login", async ({ page }) => {
    // Create a regular user, ban them, attempt login
    await page.request.post("/api/users", {
      data: { name: "Test User", email: "testuser@example.com", password: "Password123!", role: "user", status: "active" },
    });
    await page.request.put("/api/users", {
      data: { email: "testuser@example.com" },
    });
    const users = await (await page.request.get("/api/users")).json();
    const target = users.find((u: { email: string }) => u.email === "testuser@example.com");
    expect(target).toBeDefined();
    await page.request.put(`/api/users/${target.id}`, { data: { status: "banned" } });
    await page.context().clearCookies();
    await page.goto("/login");
    await page.getByLabel("Email").fill("testuser@example.com");
    await page.getByLabel("Password").fill("Password123!");
    await page.getByRole("button", { name: /sign in/i }).click();
    await expect(page.getByText(/account is banned|banned/i)).toBeVisible({ timeout: 5000 });
  });

  test("sort column by email", async ({ page }) => {
    await page.goto("/admin/users");
    const emailHeader = page.getByRole("columnheader", { name: /email/i });
    await emailHeader.click();
    await expect(emailHeader).toHaveAttribute("aria-sort", /ascending/i);
    await emailHeader.click();
    await expect(emailHeader).toHaveAttribute("aria-sort", /descending/i);
  });

  test("sort column by role", async ({ page }) => {
    await page.goto("/admin/users");
    const roleHeader = page.getByRole("columnheader", { name: /role/i });
    await roleHeader.click();
    await expect(roleHeader).toHaveAttribute("aria-sort", /ascending/i);
    await roleHeader.click();
    await expect(roleHeader).toHaveAttribute("aria-sort", /descending/i);
  });

  test("sort column by status", async ({ page }) => {
    await page.goto("/admin/users");
    const statusHeader = page.getByRole("columnheader", { name: /status/i });
    await statusHeader.click();
    await expect(statusHeader).toHaveAttribute("aria-sort", /ascending/i);
    await statusHeader.click();
    await expect(statusHeader).toHaveAttribute("aria-sort", /descending/i);
  });

  test("filter input also filters by status", async ({ page }) => {
    await page.goto("/admin/users");
    await page.getByPlaceholder(/filter|search/i).fill("banned");
    // Should show empty state since no user name/email contains "banned"
    await expect(page.getByRole("table")).toContainText(/no users? found/i);
  });
});
