import { test, expect, type Page } from "@playwright/test";

export function isMobile(): boolean {
  return test.info().project.name === "mobile";
}

export async function loginAsAdmin(page: Page) {
  // Try known admin passwords in order
  const passwords = ["SecurePass123!", "NewPass123!"];
  let loggedIn = false;
  for (const pw of passwords) {
    const loginRes = await page.request.post("/api/auth/login", {
      data: { email: "admin@example.com", password: pw },
    });
    if (loginRes.ok()) {
      loggedIn = true;
      break;
    }
  }
  if (!loggedIn) {
    // Recreate admin via setup (only works if no users exist)
    await page.request.post("/api/setup", {
      data: { email: "admin@example.com", password: "SecurePass123!" },
    });
  }
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
