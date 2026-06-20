import { test, expect } from "@playwright/test";

test("GET /sitemap.xml returns XML sitemap", async ({ page }) => {
  const response = await page.request.get("/sitemap.xml");
  expect(response.ok()).toBeTruthy();
  expect(response.headers()["content-type"]).toContain("xml");
  const text = await response.text();
  expect(text).toContain("<?xml");
  expect(text).toContain("<urlset");
  expect(text).toContain("<url>");
  expect(text).toContain("<loc>");
});

test("GET /robots.txt returns text with Allow directive", async ({ page }) => {
  const response = await page.request.get("/robots.txt");
  expect(response.ok()).toBeTruthy();
  const text = await response.text();
  expect(text).toContain("Allow: /");
});

test("GET /robots.txt includes Sitemap reference", async ({ page }) => {
  const response = await page.request.get("/robots.txt");
  expect(response.ok()).toBeTruthy();
  const text = await response.text();
  expect(text).toContain("Sitemap:");
  expect(text).toContain("sitemap.xml");
});
