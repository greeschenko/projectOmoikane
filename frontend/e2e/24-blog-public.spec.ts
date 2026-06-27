import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Blog Public Pages", () => {
  test("blog page renders heading", async ({ page }) => {
    await page.goto("/blog");
    await expect(page.getByRole("heading", { name: "Blog", exact: true })).toBeVisible();
  });

  test("blog page shows published posts", async ({ page }) => {
    await loginAsAdmin(page);
    await page.request.post("/api/blog/posts", {
      data: { title: "Public Post", slug: "public-post", content: "Public content here", status: "published" },
    });
    await page.goto("/blog");
    await expect(page.getByText("Public Post")).toBeVisible();
  });

  test("blog page does not show draft posts", async ({ page }) => {
    await loginAsAdmin(page);
    await page.request.post("/api/blog/posts", {
      data: { title: "Hidden Draft", slug: "hidden-draft", content: "Draft", status: "draft" },
    });
    await page.goto("/blog");
    await expect(page.getByText("Hidden Draft")).not.toBeVisible();
  });

  test("blog post detail page shows post content", async ({ page }) => {
    await loginAsAdmin(page);
    await page.request.post("/api/blog/posts", {
      data: { title: "Detail Page", slug: "detail-page", content: "<p>Detail content</p>", status: "published" },
    });
    await page.goto("/blog/detail-page");
    await expect(page.getByText("Detail Page")).toBeVisible();
    await expect(page.getByText("Detail content")).toBeVisible();
  });

  test("blog post detail returns 404 for unknown slug", async ({ page }) => {
    await page.goto("/blog/nonexistent-slug");
    await expect(page.getByText(/not found|404/i)).toBeVisible();
  });

  test("blog nav link is visible in public header", async ({ page }) => {
    await page.goto("/blog");
    await expect(page.getByRole("link", { name: /blog/i }).first()).toBeVisible();
  });

  test("clicking blog post navigates to detail page", async ({ page }) => {
    await loginAsAdmin(page);
    await page.request.post("/api/blog/posts", {
      data: { title: "Clickable Post", slug: "clickable-post", content: "Clickable content", status: "published" },
    });
    await page.goto("/blog");
    await page.getByText("Clickable Post").click();
    await expect(page).toHaveURL(/\/blog\/clickable-post/);
  });
});
