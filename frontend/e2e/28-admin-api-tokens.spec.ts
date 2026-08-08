import { test, expect, type Page } from "@playwright/test";
import { loginAsAdmin, waitForHydration } from "./helpers";

async function createToken(page: Page, name: string) {
  await page.goto("/admin/api-tokens");
  await waitForHydration(page, "main");
  await page.getByRole("button", { name: /new token/i }).click();
  await page.getByLabel("Name").fill(name);
  await page.getByRole("button", { name: /create/i }).last().click();
  await expect(page.getByText(/token created/i)).toBeVisible();
  const tokenText = await page.getByRole("alert").locator("code").textContent();
  return tokenText?.trim() ?? "";
}

test.describe("Admin API Tokens", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("sidebar has API Tokens nav link", async ({ page }) => {
    await expect(page.getByRole("link", { name: /api tokens/i })).toBeVisible();
  });

  test("api tokens page loads and shows heading", async ({ page }) => {
    await page.goto("/admin/api-tokens");
    await expect(page.getByRole("heading", { name: /api tokens/i })).toBeVisible();
  });

  test("can create a token and it appears in the list", async ({ page }) => {
    await createToken(page, "E2E Token");
    await expect(page.getByText("E2E Token")).toBeVisible();
  });

  test("token works as Bearer credential against the API", async ({ page, request }) => {
    const token = await createToken(page, "Bearer Token");
    expect(token).not.toBe("");
    const res = await request.get("/api/users", {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.ok()).toBeTruthy();
  });

  test("revoked token no longer works", async ({ page, request }) => {
    const token = await createToken(page, "Revoke Me");
    await page.getByRole("button", { name: /revoke revoke me/i }).click();
    await page.getByRole("button", { name: /revoke/i }).last().click();
    await expect(page.getByText("Revoke Me")).toHaveCount(0);
    const res = await request.get("/api/users", {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status()).toBe(401);
  });
});
