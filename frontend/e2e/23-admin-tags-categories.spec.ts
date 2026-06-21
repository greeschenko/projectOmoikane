import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Admin Tags", () => {
  test("tags page has create tag button", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/blog/tags");
    await expect(page.getByRole("heading", { name: /tags/i })).toBeVisible();
  });

  test("can create a tag", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/blog/tags");
    await page.getByRole("button", { name: /new tag|create/i }).click();
    await page.getByLabel(/name/i).fill("Technology");
    await page.getByLabel(/slug/i).fill("technology");
    await page.getByRole("button", { name: /save|create/i }).click();
    await expect(page.getByText("Technology", { exact: true })).toBeVisible();
  });

  test("can delete a tag", async ({ page }) => {
    await loginAsAdmin(page);
    const res = await page.request.post("/api/blog/tags", {
      data: { name: "DeleteTag", slug: "deletetag" },
    });
    const tag = await res.json();
    await page.goto("/admin/blog/tags");
    await page.getByTestId(`delete-tag-${tag.id}`).click();
    await page.getByRole("button", { name: /delete|confirm/i }).click();
    await expect(page.getByText("DeleteTag", { exact: true })).not.toBeVisible();
  });
});

test.describe("Admin Categories", () => {
  test("categories page has create category button", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/blog/categories");
    await expect(page.getByRole("heading", { name: /categories/i })).toBeVisible();
  });

  test("can create a category", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/blog/categories");
    await page.getByRole("button", { name: /new category|create/i }).click();
    await page.getByLabel(/name/i).fill("News");
    await page.getByLabel(/slug/i).fill("news");
    await page.getByRole("button", { name: /save|create/i }).click();
    await expect(page.getByText("News", { exact: true })).toBeVisible();
  });
});
