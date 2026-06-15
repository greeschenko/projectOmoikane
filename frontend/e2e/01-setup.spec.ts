import { test, expect } from "@playwright/test";

test("redirects to /setup when no users exist", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveURL("/setup");
});

test("setup form has email and password fields", async ({ page }) => {
  await page.goto("/setup");
  await expect(page.getByLabel("Email")).toBeVisible();
  await expect(page.getByLabel(/^Password\b/)).toBeVisible();
  await expect(page.getByLabel("Confirm Password")).toBeVisible();
  await expect(page.getByRole("button", { name: /create|setup|initialize/i })).toBeVisible();
});

test("shows validation error for invalid email", async ({ page }) => {
  await page.goto("/setup");
  await page.getByLabel("Email").fill("not-an-email");
  await page.getByLabel(/^Password\b/).fill("Password123!");
  await page.getByLabel("Confirm Password").fill("Password123!");
  await page.getByRole("button", { name: /create|setup|initialize/i }).click();
  await expect(page.getByText(/valid email/i)).toBeVisible();
});

test("shows validation error when passwords do not match", async ({ page }) => {
  await page.goto("/setup");
  await page.getByLabel("Email").fill("admin@example.com");
  await page.getByLabel(/^Password\b/).fill("Password123!");
  await page.getByLabel("Confirm Password").fill("DifferentPass1!");
  await page.getByRole("button", { name: /create|setup|initialize/i }).click();
  await expect(page.getByText(/passwords? do not match|passwords? must match/i)).toBeVisible();
});

test("submits setup form and redirects to /login", async ({ page }) => {
  await page.goto("/setup");
  await page.getByLabel("Email").fill("admin@example.com");
  await page.getByLabel(/^Password\b/).fill("SecurePass123!");
  await page.getByLabel("Confirm Password").fill("SecurePass123!");
  await page.getByRole("button", { name: /create|setup|initialize/i }).click();
  await expect(page).toHaveURL("/login");
});

test("redirects to /login when setup already completed", async ({ page, request }) => {
  // Ensure a user exists (setup may already be completed from prior test)
  await request.post("/api/setup", {
    data: { email: "admin@example.com", password: "SecurePass123!" },
  });

  await page.goto("/setup");
  await page.waitForURL((url) => !url.pathname.includes("/setup"), { timeout: 10000 });
  await expect(page).toHaveURL("/login");
});
