import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Structured Data & Meta Tags", () => {
  test("public page has LD+JSON structured data", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/pages/testpage1");
    await expect(page.locator('script[type="application/ld+json"]')).toBeAttached();
  });

  test("LD+JSON contains WebSite schema", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/pages/testpage1");
    const json = await page.locator('script[type="application/ld+json"]').textContent();
    const data = JSON.parse(json || "{}");
    expect(data["@context"]).toBe("https://schema.org");
    expect(data["@type"]).toBe("WebSite");
  });

  test("public page has OG meta tags", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/pages/testpage1");
    await expect(page.locator('meta[property="og:title"]')).toBeAttached();
    await expect(page.locator('meta[property="og:description"]')).toBeAttached();
  });

  test("home page has OG meta tags", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/");
    await expect(page.locator('meta[property="og:title"]')).toBeAttached();
    await expect(page.locator('meta[property="og:description"]')).toBeAttached();
  });

  test("public page has Twitter card meta tags", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/pages/testpage1");
    await expect(page.locator('meta[name="twitter:card"]')).toBeAttached();
    await expect(page.locator('meta[name="twitter:title"]')).toBeAttached();
  });
});
