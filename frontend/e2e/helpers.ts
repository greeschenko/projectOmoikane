import { test, expect, type Page } from "@playwright/test";

// Waits until React has hydrated the element matching `selector`. Before
// hydration, native form submission or clicks land before React attaches its
// handlers (visible as a GET to "/login?" or a dialog that never opens).
// React attaches internal `__reactProps$*` expando properties during commit.
export async function waitForHydration(page: Page, selector: string): Promise<void> {
  await page.waitForFunction(
    (sel) => {
      const el = document.querySelector(sel);
      if (!el) return false;
      return Object.keys(el).some((k) => k.startsWith("__reactProps$"));
    },
    selector,
    { timeout: 15000 }
  );
}

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
    // Login with the password we just set
    await page.request.post("/api/auth/login", {
      data: { email: "admin@example.com", password: "SecurePass123!" },
    });
  }
  await page.goto("/admin", { waitUntil: "domcontentloaded" });
  await expect(page).toHaveURL(/\/admin/, { timeout: isMobile() ? 15000 : 10000 });
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
