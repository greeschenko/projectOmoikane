import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Admin Tags", () => {
  test("tags tab shows create tag button", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/blog");
    await page.getByRole("tab", { name: /tags/i }).click();
    await expect(page.getByRole("button", { name: /new tag/i })).toBeVisible();
  });

  test("can create a tag via tab", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/blog");
    await page.getByRole("tab", { name: /tags/i }).click();
    await page.getByRole("button", { name: /new tag/i }).click();
    await page.getByLabel(/name/i).fill("Technology");
    await page.getByRole("button", { name: /create/i }).click();
    await expect(page.getByText("Tag created")).toBeVisible();
  });

  test("can delete a tag via tab", async ({ page }) => {
    await loginAsAdmin(page);
    await page.request.post("/api/blog/tags", {
      data: { name: "DeleteTag", slug: "deletetag" },
    });
    await page.goto("/admin/blog");
    await page.getByRole("tab", { name: /tags/i }).click();
    await page.locator('[data-testid="DeleteIcon"]').first().click();
    await page.getByRole("button", { name: /delete/i }).last().click();
    await expect(page.getByText("Tag deleted")).toBeVisible();
  });
});

test.describe("Admin Categories", () => {
  test("categories tab shows create category button", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/blog");
    await page.getByRole("tab", { name: /categories/i }).click();
    await expect(page.getByRole("button", { name: /new category/i })).toBeVisible();
  });

  test("can create a category via tab", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/blog");
    await page.getByRole("tab", { name: /categories/i }).click();
    await page.getByRole("button", { name: /new category/i }).click();
    await page.getByLabel(/name/i).fill("News");
    await page.getByRole("button", { name: /create/i }).click();
    await expect(page.getByText("Category created")).toBeVisible();
  });
});
