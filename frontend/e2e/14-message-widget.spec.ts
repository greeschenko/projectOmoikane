import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Message Widget", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    await page.request.delete("/api/messages");
  });

  test("Admin AppBar has a message bell icon", async ({ page }) => {
    await page.goto("/admin");
    const header = page.getByRole("banner", { name: /admin header|admin appbar|appbar/i }).first();
    await expect(header.getByLabel(/notifications/i)).toBeVisible();
  });

  test("bell icon shows unread count badge", async ({ page }) => {
    await page.request.post("/api/messages", {
      data: { title: "Welcome to Omoikane", content: "Thanks for joining!" },
    });
    await page.goto("/admin");
    const bell = page.getByLabel(/notifications/i);
    await expect(bell).toContainText(/1/);
  });

  test("clicking bell opens dropdown with message list", async ({ page }) => {
    await page.request.post("/api/messages", {
      data: { title: "System Update", content: "Server maintenance tonight." },
    });
    await page.goto("/admin");
    await page.getByLabel(/notifications/i).click();
    const menu = page.getByRole("menu");
    await expect(menu).toBeVisible();
    await expect(menu.getByText(/system update/i)).toBeVisible();
  });

  test("unread count decreases after marking all as read", async ({ page }) => {
    await page.request.post("/api/messages", {
      data: { title: "Read Me", content: "Please read this." },
    });
    await page.goto("/admin");
    const bell = page.getByLabel(/notifications/i);
    await expect(bell).toContainText(/1/);
    await bell.click();
    await page.getByRole("button", { name: /mark all as read/i }).click();
    // MUI Badge hides with CSS when badgeContent=0 but text remains in DOM
    await expect(bell.locator(".MuiBadge-badge")).not.toBeVisible({ timeout: 10000 });
  });

  test("message dropdown has Mark all as read action", async ({ page }) => {
    await page.request.post("/api/messages", {
      data: { title: "First", content: "First message" },
    });
    await page.request.post("/api/messages", {
      data: { title: "Second", content: "Second message" },
    });
    await page.goto("/admin");
    await page.getByLabel(/notifications/i).click();
    await expect(page.getByRole("menu")).toBeVisible();
    await expect(page.getByRole("menu").getByRole("button", { name: /mark all as read/i })).toBeVisible();
  });

  test("new system message creates notification", async ({ page }) => {
    await page.request.post("/api/messages", {
      data: { title: "New user registered", content: "A new user has joined." },
    });
    await page.goto("/admin");
    await page.getByLabel(/notifications/i).click();
    const menu = page.getByRole("menu");
    await expect(menu.getByText(/new user registered/i)).toBeVisible();
  });

  test("message bell is accessible from public header when logged in", async ({ page }) => {
    await page.goto("/pages/testpage1");
    const banner = page.getByRole("banner", { name: /public/i });
    await expect(banner.getByLabel(/notifications/i)).toBeVisible();
  });
});
