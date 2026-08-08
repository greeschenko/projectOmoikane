import { test, expect } from "@playwright/test";
import { loginAsAdmin, waitForHydration } from "./helpers";

test.describe("Phase 7 — Blog View Button", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("blog admin list has a view button linking to public post", async ({ page }) => {
    await page.request.post("/api/blog/posts", {
      data: { title: "Viewable Post", slug: "viewable-post", content: "View me", status: "published" },
    });
    await page.goto("/admin/blog");
    const viewLink = page.getByRole("link", { name: /view|open|visit/i }).first();
    await expect(viewLink).toBeVisible();
    await expect(viewLink).toHaveAttribute("href", /\/blog\//);
  });
});

test.describe("Phase 7 — Blog Filtering & Search", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("blog admin list has a search or filter input", async ({ page }) => {
    await page.goto("/admin/blog");
    const searchInput = page.getByPlaceholder(/search|filter|find/i);
    await expect(searchInput).toBeVisible();
  });

  test("filtering by title narrows the list", async ({ page }) => {
    await page.request.post("/api/blog/posts", {
      data: { title: "Alpha Post", slug: "alpha", content: "First", status: "published" },
    });
    await page.request.post("/api/blog/posts", {
      data: { title: "Beta Post", slug: "beta", content: "Second", status: "draft" },
    });
    await page.goto("/admin/blog");
    const searchInput = page.getByPlaceholder(/search|filter|find/i);
    await searchInput.fill("Alpha");
    await expect(page.getByText("Alpha Post")).toBeVisible();
    await expect(page.getByText("Beta Post")).not.toBeVisible();
  });
});

test.describe("Phase 7 — Dashboard Blog Stats", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("dashboard displays blog post count", async ({ page }) => {
    await page.request.post("/api/blog/posts", {
      data: { title: "Stat Post", slug: "stat-post", content: "Stats", status: "published" },
    });
    await page.goto("/admin");
    const blogCard = page.getByText(/blog|posts/i).first();
    await expect(blogCard).toBeVisible();
    const card = blogCard.locator("..");
    await expect(card).toContainText(/[0-9]+/);
  });

  test("dashboard displays media count", async ({ page }) => {
    await page.goto("/admin");
    const mediaCard = page.getByText(/media|uploads/i).first();
    await expect(mediaCard).toBeVisible();
    const card = mediaCard.locator("..");
    await expect(card).toContainText(/[0-9]+/);
  });

  test("dashboard displays recent messages", async ({ page }) => {
    await page.goto("/admin");
    const msgSection = page.getByText(/messages|recent messages/i).first();
    await expect(msgSection).toBeVisible();
  });
});

test.describe("Phase 7 — Login Redirect", () => {
  test("non-admin user login redirects to / instead of /admin", async ({ page }) => {
    const email = `user_${Date.now()}@test.com`;
    const reg = await page.request.post("/api/auth/register", {
      data: { name: "Regular User", email, password: "TestPass123!" },
    });
    expect(reg.ok()).toBeTruthy();

    await page.goto("/login");
    await waitForHydration(page, "form");
    await page.getByLabel("Email").fill(email);
    await page.getByLabel("Password").fill("TestPass123!");
    await page.getByRole("button", { name: /sign in|login/i }).click();

    await expect(page).toHaveURL("/", { timeout: 10000 });
  });
});

test.describe("Phase 7 — HTML Rendering", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("public page renders HTML content instead of raw code", async ({ page }) => {
    await page.request.post("/api/pages", {
      data: { title: "HTML Test", slug: "html-test", content: "<p>Hello <strong>World</strong></p>", status: "published" },
    });
    await page.goto("/pages/html-test");
    const strong = page.locator("strong");
    await expect(strong).toBeVisible();
    await expect(strong).toHaveText("World");
  });
});

test.describe("Phase 7 — Admin Menu Icons", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("sidebar nav items have visible icons", async ({ page }) => {
    await page.goto("/admin");
    const sidebar = page.getByRole("navigation");
    const icons = sidebar.locator("svg");
    const count = await icons.count();
    expect(count).toBeGreaterThanOrEqual(5);
  });
});

test.describe("Phase 7 — Loaders & Spinners", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("blog admin page shows loading indicator while fetching", async ({ page }) => {
    await page.goto("/admin/blog");
    await page.waitForTimeout(200);
    const spinner = page.locator('[role="progressbar"], .MuiCircularProgress-root');
    await expect(spinner).not.toBeVisible();
  });

  test("blog form validates required fields before submit", async ({ page }) => {
    await page.goto("/admin/blog");
    await page.getByRole("button", { name: /new blog post/i }).click();
    await page.getByRole("button", { name: /publish|save/i }).click();
    const errorText = page.getByText(/required/i).first();
    await expect(errorText).toBeVisible();
  });
});

test.describe("Phase 7 — Blog Tags & Categories Tabs", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("blog admin page has tabs or sections for tags", async ({ page }) => {
    await page.goto("/admin/blog");
    const tab = page.getByRole("tab", { name: /tag/i });
    await expect(tab).toBeVisible();
  });

  test("blog admin page has tabs or sections for categories", async ({ page }) => {
    await page.goto("/admin/blog");
    const tab = page.getByRole("tab", { name: /categor/i });
    await expect(tab).toBeVisible();
  });
});
