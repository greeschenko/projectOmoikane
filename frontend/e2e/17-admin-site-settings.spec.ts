import { test, expect } from "@playwright/test";
import { isMobile, loginAsAdmin } from "./helpers";

test("redirects to /login when not authenticated", async ({ page }) => {
  await page.goto("/admin/settings");
  await expect(page).toHaveURL("/login");
});

test.describe("Site Settings", () => {
  test("sidebar has Settings nav link", async ({ page }) => {
    test.skip(isMobile(), "Sidebar is collapsed behind hamburger on mobile");
    await loginAsAdmin(page);
    await page.goto("/admin");
    const sidebar = page.locator("nav, aside, [role='navigation']").first();
    await expect(sidebar.getByRole("link", { name: /settings/i })).toBeVisible();
  });

  test("settings page has site name field", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/settings");
    await expect(page.getByLabel(/site name/i)).toBeVisible();
  });

  test("settings page has tagline field", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/settings");
    await expect(page.getByLabel(/tagline|site description/i)).toBeVisible();
  });

  test("settings page has logo upload field", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/settings");
    await expect(page.getByRole("button", { name: "logo" })).toBeVisible();
  });

  test("settings page has favicon upload field", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/settings");
    await expect(page.getByRole("button", { name: "favicon" })).toBeVisible();
  });

  test("save button persists settings across reload", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/settings");
    const nameInput = page.getByLabel(/site name/i);
    await nameInput.fill("My Custom CMS");
    const taglineInput = page.getByLabel(/tagline|site description/i);
    await taglineInput.fill("A great platform");
    await page.getByRole("button", { name: /save|update/i }).click();
    await expect(page.getByText(/saved|updated|success/i)).toBeVisible();

    await page.reload();
    await expect(page.getByLabel(/site name/i)).toHaveValue("My Custom CMS");
    await expect(page.getByLabel(/tagline|site description/i)).toHaveValue("A great platform");
  });

  test("public header shows site name from settings", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/settings");
    await page.getByLabel(/site name/i).fill("Branding Test");
    await page.getByRole("button", { name: /save|update/i }).click();
    await expect(page.getByText(/saved|updated|success/i)).toBeVisible();

    await page.goto("/");
    await expect(page.getByRole("heading", { name: /branding test/i })).toBeVisible();
  });

  test("footer shows site name from settings", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/settings");
    await page.getByLabel(/site name/i).fill("Footer Test Inc");
    await page.getByRole("button", { name: /save|update/i }).click();
    await expect(page.getByText(/saved|updated|success/i)).toBeVisible();

    await page.goto("/");
    await expect(page.locator("footer")).toContainText(/footer test inc/i);
  });

  test("admin header shows site name from settings", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/settings");
    await page.getByLabel(/site name/i).fill("Admin Brand");
    await page.getByRole("button", { name: /save|update/i }).click();
    await expect(page.getByText(/saved|updated|success/i)).toBeVisible();

    await page.goto("/admin");
    await expect(page.getByRole("heading", { name: /admin brand/i })).toBeVisible();
  });
});
