import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Blog RSS Feed", () => {
  test("RSS feed returns valid XML", async ({ page }) => {
    await loginAsAdmin(page);
    await page.request.post("/api/blog/posts", {
      data: { title: "RSS Post", slug: "rss-post", content: "RSS content", status: "published" },
    });
    const res = await page.request.get("/rss");
    expect(res.ok()).toBeTruthy();
    expect(res.headers()["content-type"]).toContain("xml");
    const text = await res.text();
    expect(text).toContain("<?xml");
    expect(text).toContain("<rss");
    expect(text).toContain("RSS Post");
  });

  test("RSS feed does not include draft posts", async ({ page }) => {
    await loginAsAdmin(page);
    await page.request.post("/api/blog/posts", {
      data: { title: "Draft RSS", slug: "draft-rss", content: "Hidden", status: "draft" },
    });
    const res = await page.request.get("/rss");
    const text = await res.text();
    expect(text).not.toContain("Draft RSS");
  });

  test("RSS feed has channel title", async ({ page }) => {
    const res = await page.request.get("/rss");
    const text = await res.text();
    expect(text).toContain("<title>");
    expect(text).toContain("</title>");
  });
});

test.describe("User Liked Posts", () => {
  test("like count appears on blog detail page", async ({ page }) => {
    await loginAsAdmin(page);
    const res = await page.request.post("/api/blog/posts", {
      data: { title: "Liked Post", slug: "liked-post", content: "Liked", status: "published" },
    });
    const post = await res.json();
    await page.request.post(`/api/blog/posts/${post.id}/like`);
    await page.goto("/blog/liked-post");
    await expect(page.getByText(/1 like|❤|likes/i)).toBeVisible();
  });
});
