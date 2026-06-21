import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Blog API", () => {
  test("POST /api/blog/posts creates a blog post", async ({ page }) => {
    await loginAsAdmin(page);
    const res = await page.request.post("/api/blog/posts", {
      data: {
        title: "My First Post",
        slug: "my-first-post",
        content: "<p>Hello world</p>",
        status: "published",
      },
    });
    expect(res.ok()).toBeTruthy();
    const post = await res.json();
    expect(post.title).toBe("My First Post");
    expect(post.slug).toBe("my-first-post");
    expect(post.authorId).toBeDefined();
  });

  test("GET /api/blog/posts returns list of posts", async ({ page }) => {
    await loginAsAdmin(page);
    await page.request.post("/api/blog/posts", {
      data: { title: "Post A", slug: "post-a", content: "A", status: "published" },
    });
    await page.request.post("/api/blog/posts", {
      data: { title: "Post B", slug: "post-b", content: "B", status: "draft" },
    });
    const res = await page.request.get("/api/blog/posts");
    expect(res.ok()).toBeTruthy();
    const posts = await res.json();
    expect(posts.length).toBeGreaterThanOrEqual(2);
  });

  test("GET /api/blog/posts/:id returns a single post", async ({ page }) => {
    await loginAsAdmin(page);
    const created = await (await page.request.post("/api/blog/posts", {
      data: { title: "Detail Post", slug: "detail-post", content: "Detail", status: "published" },
    })).json();
    const res = await page.request.get(`/api/blog/posts/${created.id}`);
    expect(res.ok()).toBeTruthy();
    const post = await res.json();
    expect(post.title).toBe("Detail Post");
  });

  test("PUT /api/blog/posts/:id updates a blog post", async ({ page }) => {
    await loginAsAdmin(page);
    const created = await (await page.request.post("/api/blog/posts", {
      data: { title: "Original", slug: "original", content: "Old", status: "published" },
    })).json();
    const res = await page.request.put(`/api/blog/posts/${created.id}`, {
      data: { title: "Updated" },
    });
    expect(res.ok()).toBeTruthy();
    const post = await res.json();
    expect(post.title).toBe("Updated");
  });

  test("DELETE /api/blog/posts/:id deletes a blog post", async ({ page }) => {
    await loginAsAdmin(page);
    const created = await (await page.request.post("/api/blog/posts", {
      data: { title: "Delete Me", slug: "delete-me", content: "Bye", status: "published" },
    })).json();
    const del = await page.request.delete(`/api/blog/posts/${created.id}`);
    expect(del.ok()).toBeTruthy();
    const get = await page.request.get(`/api/blog/posts/${created.id}`);
    expect(get.status()).toBe(404);
  });

  test("POST /api/blog/posts/:id/like toggles like", async ({ page }) => {
    await loginAsAdmin(page);
    const created = await (await page.request.post("/api/blog/posts", {
      data: { title: "Likeable", slug: "likeable", content: "Like me", status: "published" },
    })).json();

    const like1 = await (await page.request.post(`/api/blog/posts/${created.id}/like`)).json();
    expect(like1.liked).toBe(true);
    expect(like1.count).toBe(1);

    const like2 = await (await page.request.post(`/api/blog/posts/${created.id}/like`)).json();
    expect(like2.liked).toBe(false);
    expect(like2.count).toBe(0);
  });

  test("GET /api/blog/posts/:id returns like count", async ({ page }) => {
    await loginAsAdmin(page);
    const created = await (await page.request.post("/api/blog/posts", {
      data: { title: "Counted", slug: "counted", content: "Count", status: "published" },
    })).json();
    await page.request.post(`/api/blog/posts/${created.id}/like`);

    const res = await page.request.get(`/api/blog/posts/${created.id}`);
    const post = await res.json();
    expect(post.likeCount).toBe(1);
  });

  test("blog posts appear in sitemap.xml", async ({ page }) => {
    await loginAsAdmin(page);
    await page.request.post("/api/blog/posts", {
      data: { title: "Sitemap Post", slug: "sitemap-post", content: "SEO", status: "published" },
    });
    const res = await page.request.get("/sitemap.xml");
    const text = await res.text();
    expect(text).toContain("sitemap-post");
  });
});
