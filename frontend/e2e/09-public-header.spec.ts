import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Public Header", () => {
  test("shows on the home page", async ({ page }) => {
    await page.request.post("/api/setup", {
      data: { email: "admin@example.com", password: "SecurePass123!" },
    });
    await page.goto("/");
    await expect(page.getByRole("banner", { name: /public/i })).toBeVisible();
  });

  test("shows on public pages", async ({ page }) => {
    await loginAsAdmin(page);
    await page.context().clearCookies();
    await page.goto("/pages/testpage1");
    await expect(page.getByRole("banner", { name: /public/i })).toBeVisible();
  });

  test("shows Login button when not authenticated", async ({ page }) => {
    await page.request.post("/api/setup", {
      data: { email: "admin@example.com", password: "SecurePass123!" },
    });
    await page.request.post("/api/auth/logout");
    await page.context().clearCookies();
    await page.goto("/");
    await expect(
      page.getByRole("banner", { name: /public/i }).getByRole("link", { name: /login/i })
    ).toBeVisible();
  });

  test("Login button navigates to /login", async ({ page }) => {
    await page.request.post("/api/setup", {
      data: { email: "admin@example.com", password: "SecurePass123!" },
    });
    await page.request.post("/api/auth/logout");
    await page.context().clearCookies();
    await page.goto("/");
    await page.getByRole("banner", { name: /public/i }).getByRole("link", { name: /login/i }).click();
    await expect(page).toHaveURL("/login");
  });
});

test.describe("Authenticated Header", () => {
  test("shows user menu when logged in", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/");
    const trigger = page.getByRole("banner", { name: /public/i }).getByLabel("user menu");
    await expect(trigger).toBeVisible();
    await trigger.click();
    await expect(page.getByRole("menu")).toBeVisible();
  });

  test("menu has Settings and Exit for regular user", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/");
    const trigger = page.getByRole("banner", { name: /public/i }).getByLabel("user menu");
    await trigger.click();
    await expect(page.getByRole("menuitem", { name: /settings/i })).toBeVisible();
    await expect(page.getByRole("menuitem", { name: /exit|logout/i })).toBeVisible();
  });

  test("menu has Admin Panel for admin", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/");
    await page.getByRole("banner", { name: /public/i }).getByLabel("user menu").click();
    await expect(page.getByRole("menuitem", { name: /admin panel|panel|admin/i })).toBeVisible();
  });

  test("Exit logs out and shows Login button", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/");
    await page.getByRole("banner", { name: /public/i }).getByLabel("user menu").click();
    await page.getByRole("menuitem", { name: /exit|logout/i }).click();
    await page.waitForURL("/");
    const banner = page.getByRole("banner", { name: /public/i });
    await expect(banner).toBeVisible();
    await expect(banner.getByRole("link", { name: /login/i })).toBeVisible({ timeout: 5000 });
  });
});

test.describe("Header exclusion", () => {
  test("is not visible on /login", async ({ page }) => {
    await page.goto("/login");
    await expect(page.getByRole("banner", { name: /public/i })).toHaveCount(0);
  });

  test("is not visible on /register", async ({ page }) => {
    await page.goto("/register");
    await expect(page.getByRole("banner", { name: /public/i })).toHaveCount(0);
  });

  test("is not visible on /forgot-password", async ({ page }) => {
    await page.goto("/forgot-password");
    await expect(page.getByRole("banner", { name: /public/i })).toHaveCount(0);
  });

  test("is not visible on /setup", async ({ page }) => {
    await page.goto("/setup");
    await expect(page.getByRole("banner", { name: /public/i })).toHaveCount(0);
  });

  test("is not visible on admin pages", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin");
    await expect(page.getByRole("banner", { name: /public/i })).toHaveCount(0);
  });
});

test.describe("Main Menu Widget", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    // Create menu pages
    const existing = await (await page.request.get("/api/pages")).json();
    for (const p of existing) {
      await page.request.delete(`/api/pages/${p.id}`);
    }
    await page.request.post("/api/pages", {
      data: { title: "About", slug: "about", content: "About us", status: "published", inMenu: true },
    });
    await page.context().clearCookies();
    await page.goto("/");
  });

  test("shows navigation links for pages with inMenu enabled", async ({ page }) => {
    const banner = page.getByRole("banner", { name: /public/i });
    await expect(banner.getByRole("link", { name: /about/i })).toBeVisible();
  });

  test("menu links are inside a navigation landmark", async ({ page }) => {
    const banner = page.getByRole("banner", { name: /public/i });
    const nav = banner.getByRole("navigation");
    await expect(nav).toBeVisible();
  });
});

test.describe("Message widget on main page", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    await page.request.delete("/api/messages");
  });

  test("bell icon is visible on the main page when logged in", async ({ page }) => {
    await page.goto("/");
    const banner = page.getByRole("banner", { name: /public/i });
    await expect(banner.getByLabel(/notifications/i)).toBeVisible();
  });

  test("bell shows unread count badge on the main page", async ({ page }) => {
    await page.request.post("/api/messages", {
      data: { title: "Test notification", content: "Test content" },
    });
    await page.goto("/");
    const bell = page.getByRole("banner", { name: /public/i }).getByLabel(/notifications/i);
    await expect(bell).toContainText(/1/);
  });

  test("clicking bell opens message dropdown on the main page", async ({ page }) => {
    await page.request.post("/api/messages", {
      data: { title: "System message", content: "Important update" },
    });
    await page.goto("/");
    await page.getByRole("banner", { name: /public/i }).getByLabel(/notifications/i).click();
    await expect(page.getByRole("menu")).toBeVisible();
    await expect(page.getByText(/system message/i)).toBeVisible();
  });
});
