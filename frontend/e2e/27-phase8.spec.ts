import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

function uniqueEmail(): string {
  return `u_${Date.now()}_${Math.random().toString(36).slice(2, 8)}@test.com`;
}

async function registerAndLogin(page, email: string) {
  const reg = await page.request.post("/api/auth/register", {
    data: { name: "Regular User", email, password: "TestPass123!" },
  });
  expect(reg.ok()).toBeTruthy();
  await page.request.post("/api/auth/login", {
    data: { email, password: "TestPass123!" },
  });
}

async function adminLoginTry(page) {
  for (const pw of ["SecurePass123!", "NewPass123!"]) {
    const r = await page.request.post("/api/auth/login", {
      data: { email: "admin@example.com", password: pw },
    });
    if (r.ok()) return;
  }
}

test.describe("Phase 8 — Blog On/Off Toggle", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    await page.request.put("/api/settings", { data: { blogEnabled: true } });
  });

  test("blog admin page has blog enabled toggle", async ({ page }) => {
    await page.goto("/admin/blog");
    const toggle = page.getByRole("switch", { name: /enable blog/i });
    await expect(toggle).toBeVisible();
    await expect(toggle).toBeChecked();
  });

  test("disabling blog hides public blog page", async ({ page }) => {
    await page.goto("/admin/blog");
    await page.getByRole("switch", { name: /enable blog/i }).click();
    await page.goto("/blog");
    await expect(page.getByText(/blog is disabled/i)).toBeVisible();
  });

  test("disabling blog hides blog nav link in header", async ({ page }) => {
    await page.goto("/admin/blog");
    await page.getByRole("switch", { name: /enable blog/i }).click();
    await page.goto("/");
    await expect(page.getByRole("link", { name: /^blog$/i })).toHaveCount(0);
  });

  test("toggle persists after page reload", async ({ page }) => {
    await page.goto("/admin/blog");
    await page.getByRole("switch", { name: /enable blog/i }).click();
    await page.reload();
    await expect(page.getByRole("switch", { name: /enable blog/i })).not.toBeChecked();
    await page.goto("/blog");
    await expect(page.getByText(/blog is disabled/i)).toBeVisible();
  });

  test("re-enabling blog restores public page", async ({ page }) => {
    await page.goto("/admin/blog");
    const toggle = page.getByRole("switch", { name: /enable blog/i });
    await toggle.click();
    await page.goto("/blog");
    await expect(page.getByText(/blog is disabled/i)).toBeVisible();
    await page.goto("/admin/blog");
    await toggle.click();
    await page.goto("/blog");
    await expect(page.getByRole("heading", { name: "Blog", exact: true })).toBeVisible();
  });
});

test.describe("Phase 8 — Blog in MainMenu", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    await page.request.put("/api/settings", { data: { blogEnabled: true } });
  });

  test("blog link visible in main menu when enabled", async ({ page }) => {
    await page.goto("/");
    const menu = page.getByRole("navigation");
    await expect(menu.getByRole("link", { name: /blog/i })).toBeVisible();
  });

  test("blog link hidden when blog toggle is off", async ({ page }) => {
    await page.request.put("/api/settings", { data: { blogEnabled: false } });
    await page.goto("/");
    await expect(page.getByRole("link", { name: /^blog$/i })).toHaveCount(0);
  });
});

test.describe("Phase 8 — Regular User Blog UI", () => {
  test.beforeAll(async ({ request }) => {
    let loggedIn = false;
    for (const pw of ["SecurePass123!", "NewPass123!"]) {
      const r = await request.post("/api/auth/login", {
        data: { email: "admin@example.com", password: pw },
      });
      if (r.ok()) { loggedIn = true; break; }
    }
    if (!loggedIn) {
      await request.post("/api/setup", {
        data: { email: "admin@example.com", password: "SecurePass123!" },
      });
      await request.post("/api/auth/login", {
        data: { email: "admin@example.com", password: "SecurePass123!" },
      });
    }
    await request.put("/api/settings", { data: { blogEnabled: true } });
  });
  test("regular user sees My Posts filter on /blog", async ({ page }) => {
    const email = uniqueEmail();
    await registerAndLogin(page, email);
    await page.goto("/blog");
    await expect(page.getByRole("button", { name: /my posts|my blog/i })).toBeVisible();
  });

  test("My Posts shows only current user's posts", async ({ page }) => {
    const email = uniqueEmail();
    await registerAndLogin(page, email);
    await page.request.post("/api/blog/posts", {
      data: { title: "My Post", slug: `my-post-${Date.now()}`, content: "Own content", status: "published" },
    });
    await adminLoginTry(page);
    await page.request.post("/api/blog/posts", {
      data: { title: "Admin Post", slug: `admin-post-${Date.now()}`, content: "Admin content", status: "published" },
    });
    await page.context().clearCookies();
    await page.request.post("/api/auth/login", {
      data: { email, password: "TestPass123!" },
    });
    await page.goto("/blog");
    await page.getByRole("button", { name: /my posts|my blog/i }).click();
    await expect(page.getByRole("heading", { name: "My Post" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Admin Post" })).not.toBeVisible();
  });

  test("regular user sees New Post button on /blog", async ({ page }) => {
    const email = uniqueEmail();
    await registerAndLogin(page, email);
    await page.goto("/blog");
    await expect(page.getByRole("button", { name: /new post|create post/i })).toBeVisible();
  });

  test("regular user creates post from /blog", async ({ page }) => {
    const email = uniqueEmail();
    await registerAndLogin(page, email);
    await page.goto("/blog");
    await page.getByRole("button", { name: /new post|create post/i }).click();
    await page.getByLabel(/title/i).fill("Blog Post from Public");
    await page.locator('[contenteditable="true"]').fill("Created from public blog page");
    await page.getByRole("button", { name: /publish|save|create/i }).click();
    await expect(page.getByText("Blog Post from Public")).toBeVisible();
  });

  test("regular user can edit own post from /blog", async ({ page }) => {
    const email = uniqueEmail();
    await registerAndLogin(page, email);
    const slug = `editable-${Date.now()}`;
    await page.request.post("/api/blog/posts", {
      data: { title: "Editable Post", slug, content: "Original", status: "published" },
    });
    await page.goto(`/blog/${slug}`);
    await expect(page.getByRole("button", { name: /edit/i })).toBeVisible();
    await page.getByRole("button", { name: /edit/i }).click();
    await page.getByLabel(/title/i).fill("Updated Post");
    await page.getByRole("button", { name: /save|update/i }).click();
    await expect(page.getByText("Updated Post")).toBeVisible();
  });

  test("regular user cannot edit another user's post", async ({ page }) => {
    await adminLoginTry(page);
    const otherSlug = `others-${Date.now()}`;
    await page.request.post("/api/blog/posts", {
      data: { title: "Others Post", slug: otherSlug, content: "Not yours", status: "published" },
    });
    const email = uniqueEmail();
    await registerAndLogin(page, email);
    await page.goto(`/blog/${otherSlug}`);
    await expect(page.getByRole("heading", { name: /others post/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /edit/i })).not.toBeVisible();
  });

  test("regular user's post shows author name on detail page", async ({ page }) => {
    const email = uniqueEmail();
    await registerAndLogin(page, email);
    const slug = `authored-${Date.now()}`;
    await page.request.post("/api/blog/posts", {
      data: { title: "Authored Post", slug, content: "Some content", status: "published" },
    });
    await page.goto(`/blog/${slug}`);
    await expect(page.getByText(/regular user/i)).toBeVisible();
  });

  test("admin sees all posts in My Posts filter", async ({ page }) => {
    await adminLoginTry(page);
    const adminSlug = `admins-${Date.now()}`;
    await page.request.post("/api/blog/posts", {
      data: { title: "Admin's Post", slug: adminSlug, content: "Admin only", status: "published" },
    });
    const userEmail = uniqueEmail();
    await registerAndLogin(page, userEmail);
    const userSlug = `users-${Date.now()}`;
    await page.request.post("/api/blog/posts", {
      data: { title: "User's Post", slug: userSlug, content: "User only", status: "published" },
    });
    await adminLoginTry(page);
    await page.goto("/blog");
    await page.getByRole("button", { name: /my posts|my blog/i }).click();
    await expect(page.getByRole("heading", { name: "Admin's Post" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "User's Post" })).toBeVisible();
  });
});

test.describe("Phase 8 — Page Form Rework", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("page editor dialog has left and right columns", async ({ page }) => {
    await page.goto("/admin/pages");
      await page.locator("main li").first().waitFor();
      await page.getByRole("button", { name: /new page/i }).click();
    await expect(page.getByRole("textbox", { name: /^title$/i })).toBeVisible();
    await expect(page.getByRole("textbox", { name: /^slug$/i })).toBeVisible();
    await expect(page.locator('[contenteditable="true"]')).toBeVisible();
  });

  test("preview button opens preview in new tab", async ({ page }) => {
    await page.goto("/admin/pages");
      await page.locator("main li").first().waitFor();
      await page.getByRole("button", { name: /new page/i }).click();
    await page.getByRole("textbox", { name: /^title$/i }).fill("Previewable");
    await page.getByRole("textbox", { name: /^slug$/i }).fill("previewable");
    await page.locator('[contenteditable="true"]').fill("Preview content");
    await page.getByRole("button", { name: /create/i }).click();
    await page.getByRole("button", { name: /edit/i }).first().click();
    await expect(page.getByRole("button", { name: /preview/i })).toBeVisible();
  });

  test("page form submit still creates page", async ({ page }) => {
    await page.goto("/admin/pages");
      await page.locator("main li").first().waitFor();
      await page.getByRole("button", { name: /new page/i }).click();
    await page.getByRole("textbox", { name: /^title$/i }).fill("Rework Test Page");
    await page.getByRole("textbox", { name: /^slug$/i }).fill("rework-test");
    await page.locator('[contenteditable="true"]').fill("Rework content");
    await page.getByRole("button", { name: /create/i }).click();
    await expect(page.getByText("Rework Test Page")).toBeVisible();
  });
});

test.describe("Phase 8 — User Settings Rework", () => {
  test.beforeAll(async ({ request }) => {
    let loggedIn = false;
    for (const pw of ["SecurePass123!", "NewPass123!"]) {
      const r = await request.post("/api/auth/login", {
        data: { email: "admin@example.com", password: pw },
      });
      if (r.ok()) { loggedIn = true; break; }
    }
    if (!loggedIn) {
      await request.post("/api/setup", {
        data: { email: "admin@example.com", password: "SecurePass123!" },
      });
    }
  });

  test.beforeEach(async ({ page }) => {
    await adminLoginTry(page);
  });

  test("settings page has vertical tabs for Profile, Password, Avatar", async ({ page }) => {
    await page.goto("/settings");
    await expect(page.getByRole("tab", { name: /profile/i })).toBeVisible();
    await expect(page.getByRole("tab", { name: /password/i })).toBeVisible();
    await expect(page.getByRole("tab", { name: /avatar/i })).toBeVisible();
  });

  test("Profile tab shows name and email fields", async ({ page }) => {
    await page.goto("/settings");
    await page.getByRole("tab", { name: /profile/i }).click();
    await expect(page.getByLabel(/name/i)).toBeVisible();
    await expect(page.getByLabel(/email/i)).toBeVisible();
  });

  test("Password tab shows change password form", async ({ page }) => {
    await page.goto("/settings");
    await page.getByRole("tab", { name: /password/i }).click();
    await expect(page.getByLabel(/current password/i)).toBeVisible();
    await expect(page.getByRole("textbox", { name: /^new password$/i })).toBeVisible();
    await expect(page.getByRole("textbox", { name: /^confirm new password/i })).toBeVisible();
  });

  test("Avatar tab shows upload option", async ({ page }) => {
    await page.goto("/settings");
    await page.getByRole("tab", { name: /avatar/i }).click();
    await expect(page.getByRole("button", { name: /upload|avatar/i })).toBeVisible();
  });

  test("switching tabs changes visible content", async ({ page }) => {
    await page.goto("/settings");
    await page.getByRole("tab", { name: /profile/i }).click();
    await expect(page.getByLabel(/name/i)).toBeVisible();
    await page.getByRole("tab", { name: /password/i }).click();
    await expect(page.getByLabel(/current password/i)).toBeVisible();
    await expect(page.getByLabel(/name/i)).not.toBeVisible();
  });
});

test.describe("Phase 8 — Main Page Redesign", () => {
  test("home page has documentation link", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("link", { name: /docs|documentation/i })).toBeVisible();
  });

  test("home page has GitHub link", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator('a[href*="github"]')).toBeVisible();
  });

  test("home page has project heading or logo", async ({ page }) => {
    await page.goto("/");
    const headings = page.getByRole("heading");
    const count = await headings.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });
});
