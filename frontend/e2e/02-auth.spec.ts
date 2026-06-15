import { test, expect } from "@playwright/test";

test.describe("Registration", () => {
  test("registration form has required fields", async ({ page }) => {
    await page.goto("/register");
    await expect(page.getByLabel("Name")).toBeVisible();
    await expect(page.getByLabel("Email")).toBeVisible();
    await expect(page.getByLabel(/^Password\b/)).toBeVisible();
    await expect(page.getByLabel("Confirm Password")).toBeVisible();
    await expect(page.getByRole("button", { name: /register|sign up|create account/i })).toBeVisible();
  });

  test("shows error when passwords do not match", async ({ page }) => {
    await page.goto("/register");
    await page.getByLabel("Name").fill("Test User");
    await page.getByLabel("Email").fill("test@example.com");
    await page.getByLabel(/^Password\b/).fill("Password123!");
    await page.getByLabel("Confirm Password").fill("DifferentPass1!");
    await page.getByRole("button", { name: /register|sign up|create account/i }).click();
    await expect(page.getByText(/passwords? do not match|passwords? must match/i)).toBeVisible();
  });

  test("shows validation error for invalid email", async ({ page }) => {
    await page.goto("/register");
    await page.getByLabel("Name").fill("Test User");
    await page.getByLabel("Email").fill("bad-email");
    await page.getByLabel(/^Password\b/).fill("Password123!");
    await page.getByLabel("Confirm Password").fill("Password123!");
    await page.getByRole("button", { name: /register|sign up|create account/i }).click();
    await expect(page.getByText(/valid email/i)).toBeVisible();
  });

  test("successful registration redirects to /login", async ({ page }) => {
    await page.goto("/register");
    await page.getByLabel("Name").fill("Test User");
    await page.getByLabel("Email").fill("test@example.com");
    await page.getByLabel(/^Password\b/).fill("Password123!");
    await page.getByLabel("Confirm Password").fill("Password123!");
    await page.getByRole("button", { name: /register|sign up|create account/i }).click();
    await expect(page).toHaveURL("/login");
  });

  test("has link to login page", async ({ page }) => {
    await page.goto("/register");
    await expect(page.getByRole("link", { name: /sign in|login|already have/i })).toBeVisible();
  });
});

test.describe("Login", () => {
  test("login form has required fields", async ({ page }) => {
    await page.goto("/login");
    await expect(page.getByLabel("Email")).toBeVisible();
    await expect(page.getByLabel("Password")).toBeVisible();
    await expect(page.getByRole("button", { name: /sign in|login/i })).toBeVisible();
  });

  test("shows error for invalid email format", async ({ page }) => {
    await page.goto("/login");
    await page.getByLabel("Email").fill("not-email");
    await page.getByLabel("Password").fill("somepass");
    await page.getByRole("button", { name: /sign in|login/i }).click();
    await expect(page.getByText(/valid email/i)).toBeVisible();
  });

  test("shows error for invalid credentials", async ({ page }) => {
    await page.goto("/login");
    await page.getByLabel("Email").fill("wrong@example.com");
    await page.getByLabel("Password").fill("WrongPass123!");
    await page.getByRole("button", { name: /sign in|login/i }).click();
    await expect(page.getByText(/invalid|incorrect|not found/i)).toBeVisible();
  });

  test("successful login redirects to /admin", async ({ page, request }) => {
    // Ensure user exists via API
    await request.post("/api/setup", {
      data: { email: "admin@example.com", password: "SecurePass123!" },
    });
    await page.goto("/login");
    await page.getByLabel("Email").fill("admin@example.com");
    await page.getByLabel("Password").fill("SecurePass123!");
    await page.getByRole("button", { name: /sign in|login/i }).click();
    await expect(page).toHaveURL(/\/admin/);
  });

  test("has links to register and forgot password", async ({ page }) => {
    await page.goto("/login");
    await expect(page.getByRole("link", { name: /register|sign up|create account|create an account/i })).toBeVisible();
    await expect(page.getByRole("link", { name: /forgot|reset password/i })).toBeVisible();
  });
});

test.describe("Forgot Password", () => {
  test("forgot password form has email field", async ({ page }) => {
    await page.goto("/forgot-password");
    await expect(page.getByLabel("Email")).toBeVisible();
    await expect(page.getByRole("button", { name: /reset|send|submit/i })).toBeVisible();
  });

  test("shows success message on submission", async ({ page }) => {
    await page.goto("/forgot-password");
    await page.getByLabel("Email").fill("test@example.com");
    await page.getByRole("button", { name: /reset|send|submit/i }).click();
    await expect(page.getByText(/email sent|check your email|reset link/i)).toBeVisible();
  });

  test("has link back to login", async ({ page }) => {
    await page.goto("/forgot-password");
    await expect(page.getByRole("link", { name: /back to login|sign in/i })).toBeVisible();
  });
});
