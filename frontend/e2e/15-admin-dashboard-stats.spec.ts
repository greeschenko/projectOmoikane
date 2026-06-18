import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Dashboard Stats", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin");
  });

  test("displays user count card", async ({ page }) => {
    const userCard = page.getByText(/users|user count/i).first();
    await expect(userCard).toBeVisible();
    // Should show a number next to or inside the card
    const card = userCard.locator("..");
    await expect(card).toContainText(/[0-9]+/);
  });

  test("displays page count card", async ({ page }) => {
    const pageCard = page.getByText(/pages|page count/i).first();
    await expect(pageCard).toBeVisible();
    const card = pageCard.locator("..");
    await expect(card).toContainText(/[0-9]+/);
  });

  test("user count reflects actual number of users", async ({ page }) => {
    const users = await (await page.request.get("/api/users")).json();
    const expectedCount = users.length;
    const userCard = page.getByText(/users|user count/i).first().locator("..");
    await expect(userCard).toContainText(String(expectedCount));
  });

  test("page count reflects actual number of pages", async ({ page }) => {
    const pages = await (await page.request.get("/api/pages")).json();
    const expectedCount = pages.length;
    const pageCard = page.getByText(/pages|page count/i).first().locator("..");
    await expect(pageCard).toContainText(String(expectedCount));
  });

  test("shows a chart or graph section for user registrations", async ({ page }) => {
    const chartSection = page.getByText(/registration|registrations|new users|chart/i).first();
    await expect(chartSection).toBeVisible();
  });
});
