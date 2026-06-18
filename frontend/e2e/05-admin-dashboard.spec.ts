import { test, expect } from "@playwright/test";
import { isMobile, loginAsAdmin } from "./helpers";

test("redirects to /login when not authenticated", async ({ page }) => {
  await page.goto("/admin");
  await expect(page).toHaveURL("/login");
});

test.describe("Dashboard", () => {
  test("has sidebar with navigation links", async ({ page }) => {
    test.skip(isMobile(), "Sidebar is collapsed behind hamburger on mobile");
    await loginAsAdmin(page);
    await page.goto("/admin");
    const sidebar = page.locator("nav, aside, [role='navigation']").first();
    await expect(sidebar.getByRole("link", { name: /dashboard/i })).toBeVisible();
    await expect(sidebar.getByRole("link", { name: /users/i })).toBeVisible();
    await expect(sidebar.getByRole("link", { name: /pages/i })).toBeVisible();
  });

  test("displays dashboard heading", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin");
    await expect(page.locator("h1")).toContainText(/dashboard/i);
  });
});

test.describe("Admin Header", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin");
  });

  test("has a header AppBar with user widget", async ({ page }) => {
    const header = page.getByRole("banner", { name: /admin|appbar/i }).first();
    await expect(header).toBeVisible();
    await expect(header.getByRole("button", { name: /user|avatar|account/i })).toBeVisible();
  });

  test("header has message notification bell", async ({ page }) => {
    const header = page.getByRole("banner", { name: /admin|appbar/i }).first();
    await expect(header.getByLabel(/messages|notifications|bell/i)).toBeVisible();
  });

  test("user menu has Settings and Exit options", async ({ page }) => {
    const header = page.getByRole("banner", { name: /admin|appbar/i }).first();
    await header.getByRole("button", { name: /user|avatar|account/i }).click();
    await expect(page.getByRole("menuitem", { name: /settings/i })).toBeVisible();
    await expect(page.getByRole("menuitem", { name: /exit|logout/i })).toBeVisible();
  });
});
