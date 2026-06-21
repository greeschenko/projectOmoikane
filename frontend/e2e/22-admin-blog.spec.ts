import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Admin Blog", () => {
  test("sidebar has Blog nav link", async ({ page }) => {
    await loginAsAdmin(page);
    await expect(page.getByRole("link", { name: /blog/i })).toBeVisible();
  });

  test("blog page shows empty state", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/blog");
    await expect(page).toHaveURL(/\/admin\/blog/);
    await expect(page.getByText(/no blog posts|create your first/i)).toBeVisible();
  });

  test("new post button opens create form with title and content fields", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/blog");
    await page.getByRole("button", { name: /new blog post|create|new post/i }).click();
    await expect(page.getByLabel(/title/i)).toBeVisible();
    await expect(page.getByLabel(/content/i)).toBeVisible();
  });

  test("creating a blog post shows it in the list", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/admin/blog");
    await page.getByRole("button", { name: /new blog post|create|new post/i }).click();
    await page.getByLabel(/title/i).fill("Test Blog Post");
    await page.locator('[contenteditable="true"]').fill("Blog content here");
    await page.getByRole("button", { name: /publish|save|create/i }).click();
    await expect(page.getByText("Test Blog Post")).toBeVisible();
  });

  test("blog post status shows draft or published", async ({ page }) => {
    await loginAsAdmin(page);
    await page.request.post("/api/blog/posts", {
      data: { title: "Draft Post", slug: "draft-post", content: "Draft content", status: "draft" },
    });
    await page.goto("/admin/blog");
    await expect(page.getByText("Draft Post")).toBeVisible();
    await expect(page.getByText("draft", { exact: true }).first()).toBeVisible();
  });
});
