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

  test("blog list shows category and tag chips with a category filter", async ({ page }) => {
    await loginAsAdmin(page);
    const cat = await (
      await page.request.post("/api/blog/categories", { data: { name: "Tech", slug: "tech" } })
    ).json();
    await page.request.post("/api/blog/tags", { data: { name: "React", slug: "react" } });
    await page.request.post("/api/blog/posts", {
      data: {
        title: "Tagged Post",
        slug: "tagged-post",
        content: "Tagged content",
        status: "published",
        categoryId: cat.id,
        tags: ["React"],
      },
    });
    await page.goto("/blog");

    const card = page.locator("a[href='/blog/tagged-post']");
    await expect(card).toBeVisible();
    await expect(card.getByText("Tagged Post")).toBeVisible();
    // Category + tag chips surfaced on the list card
    await expect(card.getByText("Tech", { exact: true })).toBeVisible();
    await expect(card.getByText("React", { exact: true })).toBeVisible();
    // Category filter present (only rendered when categories exist)
    await expect(page.getByRole("combobox", { name: /category/i })).toBeVisible();

    // Filtering by Tech keeps the tagged post visible
    await page.getByRole("combobox", { name: /category/i }).click();
    await page.getByRole("option", { name: "Tech" }).click();
    await expect(page.getByText("Tagged Post")).toBeVisible();
    // A post from another category is hidden by the filter
    await expect(page.getByText("Public Post")).not.toBeVisible();
  });

  test("blog post detail shows category and tag chips", async ({ page }) => {
    await loginAsAdmin(page);
    const cat = await (
      await page.request.post("/api/blog/categories", { data: { name: "Science", slug: "science" } })
    ).json();
    await page.request.post("/api/blog/posts", {
      data: {
        title: "Chipped Post",
        slug: "chipped-post",
        content: "<p>Chipped content</p>",
        status: "published",
        categoryId: cat.id,
        tags: ["React"],
      },
    });
    await page.goto("/blog/chipped-post");
    await expect(page.getByText("Chipped Post")).toBeVisible();
    await expect(page.getByText("Science", { exact: true })).toBeVisible();
    await expect(page.getByText("React", { exact: true })).toBeVisible();
  });
});
