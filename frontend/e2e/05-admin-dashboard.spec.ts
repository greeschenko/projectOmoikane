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
