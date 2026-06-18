import { test, expect } from "@playwright/test";

async function setupPages(page: any, request: any) {
  // Ensure admin user exists
  await request.post("/api/setup", {
    data: { email: "admin@example.com", password: "SecurePass123!" },
  });
  // Login via browser to set session cookie
  await page.goto("/login");
  await page.getByLabel("Email").fill("admin@example.com");
  await page.getByLabel("Password").fill("SecurePass123!");
  await page.getByRole("button", { name: /sign in|login/i }).click();
  await expect(page).toHaveURL(/\/admin/);
  // Create page tree via API
  const p1 = await page.request.post("/api/pages", {
    data: { title: "Test Page 1", slug: "testpage1", content: "Content 1", status: "published" },
  });
  const p1data = await p1.json();
  const p2 = await page.request.post("/api/pages", {
    data: { title: "Test Page 2", slug: "testpage2", content: "Content 2", status: "published" },
  });
  const p2data = await p2.json();
  const p21 = await page.request.post("/api/pages", {
    data: { title: "Test Page 21", slug: "testpage21", content: "Content 21", parentId: p2data.id, status: "published" },
  });
  const p21data = await p21.json();
  await page.request.post("/api/pages", {
    data: { title: "Deep", slug: "deep", content: "Deep content", parentId: p21data.id, status: "published" },
  });
}

test.describe("Public static pages", () => {
  test("renders a top-level page with title and content", async ({ page, request }) => {
    await setupPages(page, request);
    await page.goto("/pages/testpage1");
    await expect(page.locator("h1")).toBeVisible();
    await expect(page.locator("main").first()).toBeVisible();
  });

  test("renders a nested child page", async ({ page, request }) => {
    await setupPages(page, request);
    await page.goto("/pages/testpage2/testpage21");
    await expect(page.locator("h1")).toBeVisible();
    await expect(page.locator("main").first()).toBeVisible();
  });

  test("renders a deeply nested page", async ({ page, request }) => {
    await setupPages(page, request);
    await page.goto("/pages/testpage2/testpage21/deep");
    await expect(page.locator("h1")).toBeVisible();
    await expect(page.locator("main").first()).toBeVisible();
  });

  test("shows page title as document title", async ({ page, request }) => {
    await setupPages(page, request);
    await page.goto("/pages/testpage1");
    const title = await page.locator("h1").textContent();
    await expect(page).toHaveTitle(new RegExp(title?.trim() ?? ""));
  });

  test("shows parent page title in breadcrumb for child page", async ({ page, request }) => {
    await setupPages(page, request);
    await page.goto("/pages/testpage2/testpage21");
    const parentLink = page.getByRole("link", { name: /test page 2/i });
    await expect(parentLink).toBeVisible();
  });

  test("shows 404 for non-existent page", async ({ page, request }) => {
    await setupPages(page, request);
    await page.goto("/pages/nonexistent-xyz");
    await expect(page.getByText(/not found|404/i)).toBeVisible();
  });
});
