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
    const existingPosts = await (await page.request.get("/api/blog/posts")).json();
    for (const post of existingPosts) {
      await page.request.delete(`/api/blog/posts/${post.id}`);
    }
    await page.reload();
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

  test.describe("Bulk actions", () => {
    test.beforeEach(async ({ page }) => {
      await loginAsAdmin(page);
      await page.request.post("/api/blog/posts", {
        data: { title: "Bulk Test Post", slug: "bulk-test-post", content: "Bulk test", status: "draft" },
      });
    });

    test("each blog post has a checkbox", async ({ page }) => {
      const postsResponse = page.waitForResponse(r => r.url().includes('/api/admin/blog/posts') && r.status() === 200);
      await page.goto("/admin/blog");
      await postsResponse;
      await expect(page.getByText("Bulk Test Post")).toBeVisible();
      const checkboxes = page.locator('input[type="checkbox"]');
      const count = await checkboxes.count();
      expect(count).toBeGreaterThanOrEqual(1);
    });

    test("bulk publish and draft posts", async ({ page }) => {
      await page.goto("/admin/blog");
      await expect(page.getByText("Bulk Test Post")).toBeVisible();
      await page.getByRole("checkbox").first().check();
      await page.getByRole("button", { name: /^publish$/i }).click();
      await expect(page.getByRole("dialog")).toBeVisible();
      await page.getByRole("dialog").getByRole("button", { name: /confirm/i }).click();
      await expect(page.getByRole("dialog")).not.toBeVisible({ timeout: 5000 });
      await page.waitForTimeout(500);
      await page.getByRole("checkbox").first().check();
      await page.getByRole("button", { name: /^draft$/i }).click();
      await expect(page.getByRole("dialog")).toBeVisible();
      await page.getByRole("dialog").getByRole("button", { name: /confirm/i }).click();
      await expect(page.getByRole("dialog")).not.toBeVisible({ timeout: 5000 });
    });

    test("bulk delete removes selected posts", async ({ page }) => {
      await page.goto("/admin/blog");
      await expect(page.getByText("Bulk Test Post")).toBeVisible();
      await page.getByRole("checkbox").first().check();
      await expect(page.getByRole("button", { name: /^delete selected$/i })).toBeVisible();
      await page.getByRole("button", { name: /^delete selected$/i }).click();
      await expect(page.getByRole("dialog")).toBeVisible();
      const refetchResponse = page.waitForResponse(r => r.url().includes('/api/admin/blog/posts') && r.status() === 200);
      await page.getByRole("dialog").getByRole("button", { name: /confirm/i }).click();
      await refetchResponse;
      await page.waitForTimeout(500);
      await expect(page.getByText("Bulk Test Post")).not.toBeVisible();
    });
  });
});
