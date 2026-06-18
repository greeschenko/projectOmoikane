import { test, expect, type Page } from "@playwright/test";

export function isMobile(): boolean {
  return test.info().project.name === "mobile";
}

export async function loginAsAdmin(page: Page) {
  // Create admin user if setup not already completed
  await page.request.post("/api/setup", {
    data: { email: "admin@example.com", password: "SecurePass123!" },
  });
  // Login via browser form to set session cookie properly
  await page.goto("/login", { waitUntil: "networkidle" });
  await page.getByLabel("Email").fill("admin@example.com");
  await page.getByLabel("Password").fill("SecurePass123!");
  await page.getByRole("button", { name: /sign in/i }).click();
  // Give the client-side fetch/router.push time to finish, then navigate directly
  await page.waitForTimeout(2000);
  await page.goto("/admin", { waitUntil: "networkidle" });
  await expect(page).toHaveURL(/\/admin/);
  // Reset and seed sample pages via API
  const existingPages = await page.request.get("/api/pages");
  const pagesData = await existingPages.json();
  if (Array.isArray(pagesData)) {
    for (const p of pagesData) {
      await page.request.delete(`/api/pages/${p.id}`);
    }
  }
  await page.request.post("/api/pages", {
    data: { title: "Test Page 1", slug: "testpage1", content: "Content 1", status: "published" },
  });
  await page.request.post("/api/pages", {
    data: { title: "Test Page 2", slug: "testpage2", content: "Content 2", status: "published" },
  });
}
