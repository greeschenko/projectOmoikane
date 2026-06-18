import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Public Footer", () => {
  test("shows on the home page", async ({ page, request }) => {
    await request.post("/api/setup", {
      data: { email: "admin@example.com", password: "SecurePass123!" },
    });
    await page.goto("/");
    await expect(page.getByRole("contentinfo")).toBeVisible();
  });
});

test.describe("Footer exclusion", () => {
  test("is not visible on /login", async ({ page }) => {
    await page.goto("/login");
    await expect(page.getByRole("contentinfo")).toHaveCount(0);
  });

  test("is not visible on /register", async ({ page }) => {
    await page.goto("/register");
    await expect(page.getByRole("contentinfo")).toHaveCount(0);
  });

  test("is not visible on /forgot-password", async ({ page }) => {
    await page.goto("/forgot-password");
    await expect(page.getByRole("contentinfo")).toHaveCount(0);
  });

  test("is not visible on /setup", async ({ page }) => {
    await page.goto("/setup");
    await expect(page.getByRole("contentinfo")).toHaveCount(0);
  });

  test("is not visible on admin pages", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin");
    await expect(page.getByRole("contentinfo")).toHaveCount(0);
  });
});
