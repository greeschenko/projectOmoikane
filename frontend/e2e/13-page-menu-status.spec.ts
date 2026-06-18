import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Page Status & Menu", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("page creation form has Show in menu toggle", async ({ page }) => {
    await page.goto("/admin/pages");
    await page.getByRole("button", { name: /new page|add page|create page/i }).click();
    await expect(page.getByLabel(/show in menu|in menu|menu/i)).toBeVisible();
  });

  test("page creation form has Status dropdown", async ({ page }) => {
    await page.goto("/admin/pages");
    await page.getByRole("button", { name: /new page|add page|create page/i }).click();
    await expect(page.getByRole("dialog").getByText("Status").locator("..").getByRole("combobox")).toBeVisible();
  });

  test("status defaults to Draft for new pages", async ({ page }) => {
    await page.goto("/admin/pages");
    await page.getByRole("button", { name: /new page|add page|create page/i }).click();
    const statusField = page.getByRole("dialog").getByText("Status").locator("..").getByRole("combobox");
    await expect(statusField).toHaveText(/draft/i);
  });

  test("in menu defaults to unchecked for new pages", async ({ page }) => {
    await page.goto("/admin/pages");
    await page.getByRole("button", { name: /new page|add page|create page/i }).click();
    const toggle = page.getByLabel(/show in menu|in menu|menu/i);
    await expect(toggle).not.toBeChecked();
  });

  test("can create a published page with inMenu enabled", async ({ page }) => {
    await page.goto("/admin/pages");
    await page.getByRole("button", { name: /new page|add page|create page/i }).click();
    await page.getByLabel(/^Title\b/i).fill("Menu Page");
    await page.getByLabel("Slug").fill("menu-page");
    await page.getByLabel("Content").fill("Menu page content");
    await page.getByRole("dialog").getByText("Status").locator("..").getByRole("combobox").click();
    await page.getByRole("option", { name: /published/i }).click();
    await page.getByLabel(/show in menu|in menu|menu/i).click();
    await page.getByRole("button", { name: /save|create|submit/i }).click();
    await expect(page.locator("main ul")).toContainText(/menu page/i);
  });

  test("page tree has View button that opens in new tab", async ({ page }) => {
    await page.goto("/admin/pages");
    const firstItem = page.locator("main li").first();
    const viewButton = firstItem.getByRole("button", { name: /view|open|eye/i });
    await expect(viewButton).toBeVisible();
    const href = await viewButton.getAttribute("data-href");
    expect(href).toBeTruthy();
  });
});

test.describe("Public Header Menu Widget", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    const existing = await (await page.request.get("/api/pages")).json();
    for (const p of existing) {
      await page.request.delete(`/api/pages/${p.id}`);
    }
    await page.request.post("/api/pages", {
      data: { title: "About", slug: "about", content: "About us", status: "published", inMenu: true },
    });
    await page.request.post("/api/pages", {
      data: { title: "Services", slug: "services", content: "Our services", status: "published", inMenu: true },
    });
    await page.request.post("/api/pages", {
      data: { title: "Hidden", slug: "hidden", content: "Hidden page", status: "published", inMenu: false },
    });
    await page.request.post("/api/pages", {
      data: { title: "Draft Page", slug: "draft", content: "Draft page", status: "draft", inMenu: true },
    });
  });

  test("header shows menu with pages that have inMenu enabled", async ({ page }) => {
    await page.context().clearCookies();
    await page.goto("/pages/about");
    const banner = page.getByRole("banner", { name: /public/i });
    await expect(banner.getByRole("link", { name: /about/i })).toBeVisible();
    await expect(banner.getByRole("link", { name: /services/i })).toBeVisible();
  });

  test("header menu does not show pages with inMenu disabled", async ({ page }) => {
    await page.context().clearCookies();
    await page.goto("/pages/about");
    const banner = page.getByRole("banner", { name: /public/i });
    await expect(banner.getByRole("link", { name: /hidden/i })).toHaveCount(0);
  });

  test("header menu does not show draft pages even if inMenu is enabled", async ({ page }) => {
    await page.context().clearCookies();
    await page.goto("/pages/about");
    const banner = page.getByRole("banner", { name: /public/i });
    await expect(banner.getByRole("link", { name: /draft/i })).toHaveCount(0);
  });

  test("menu links navigate to the correct page", async ({ page }) => {
    await page.context().clearCookies();
    await page.goto("/pages/about");
    const banner = page.getByRole("banner", { name: /public/i });
    await banner.getByRole("link", { name: /about/i }).click();
    await expect(page).toHaveURL(/\/pages\/about/);
  });
});
