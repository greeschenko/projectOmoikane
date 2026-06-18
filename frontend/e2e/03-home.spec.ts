import { test, expect } from "@playwright/test";

test("displays the project title", async ({ page, request }) => {
  // Ensure a user exists so home page doesn't redirect to /setup
  await request.post("/api/setup", {
    data: { email: "admin@example.com", password: "SecurePass123!" },
  });
  await page.goto("/");
  await expect(page.locator("h1")).toContainText("projectOmoikane");
});

test("shows Get Started and Learn More buttons", async ({ page, request }) => {
  await request.post("/api/setup", {
    data: { email: "admin@example.com", password: "SecurePass123!" },
  });
  await page.goto("/");
  await expect(
    page.getByRole("button", { name: "Get Started" })
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Learn More" })
  ).toBeVisible();
});
