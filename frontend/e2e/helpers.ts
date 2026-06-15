import { test, expect, type Page } from "@playwright/test";

export function isMobile(): boolean {
  return test.info().project.name === "mobile";
}

export async function loginAsAdmin(page: Page) {
  // Create admin user via API request (from Node.js process)
  await page.request.post("/api/setup", {
    data: { email: "admin@example.com", password: "SecurePass123!" },
  });
  // Log in via browser to set session cookie
  await page.goto("/login", { waitUntil: "networkidle" });
  await page.getByLabel("Email").fill("admin@example.com");
  await page.getByLabel("Password").fill("SecurePass123!");
  await page.getByRole("button", { name: /sign in/i }).click();
  await page.waitForURL((url) => !url.pathname.includes("/login"), { timeout: 10000 });
  // Reset and seed sample pages via API (session cookie now set)
  const existingPages = await page.request.get("/api/pages");
  const pagesData = await existingPages.json();
  if (Array.isArray(pagesData)) {
    for (const p of pagesData) {
      await page.request.delete(`/api/pages/${p.id}`);
    }
  }
  await page.request.post("/api/pages", {
    data: { title: "Test Page 1", slug: "testpage1", content: "Content 1" },
  });
  await page.request.post("/api/pages", {
    data: { title: "Test Page 2", slug: "testpage2", content: "Content 2" },
  });
}
