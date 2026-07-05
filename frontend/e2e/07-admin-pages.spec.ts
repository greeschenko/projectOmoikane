import { test, expect } from "@playwright/test";
import { isMobile, loginAsAdmin } from "./helpers";

test.describe("Admin Pages", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test.describe("Page tree", () => {
    test("shows page tree with parent-child structure", async ({ page }) => {
      await page.goto("/admin/pages");
      const tree = page.locator("main ul").first();
      await expect(tree).toBeVisible();
    });

    test("has 'New Page' button", async ({ page }) => {
      await page.goto("/admin/pages");
      await expect(page.getByRole("button", { name: /new page|add page|create page/i })).toBeVisible();
    });

    test("each page has edit and delete actions", async ({ page }) => {
      test.skip(isMobile(), "Action buttons may be in a swipe menu on mobile");
      await page.goto("/admin/pages");
      const firstItem = page.locator("main li").first();
      await expect(firstItem.getByRole("button", { name: /edit|pencil/i })).toBeVisible();
      await expect(firstItem.getByRole("button", { name: /delete|remove|trash/i })).toBeVisible();
    });
  });

  test.describe("Page creation form", () => {
    const formFields = ["Title", "Slug", "Content"];

    test("'New Page' button opens a form", async ({ page }) => {
      await page.goto("/admin/pages");
      await page.locator("main li").first().waitFor();
      await page.getByRole("button", { name: /new page|add page|create page/i }).click();
      await expect(page.getByRole("dialog")).toBeVisible();
    });

    test("form has title, slug, content, meta fields, parent selector, status, and menu toggle", async ({ page }) => {
      await page.goto("/admin/pages");
      await page.locator("main li").first().waitFor();
      await page.getByRole("button", { name: /new page|add page|create page/i }).click();
      await expect(page.getByLabel(/^Title\b/i)).toBeVisible();
      await expect(page.getByLabel("Slug")).toBeVisible();
      await expect(page.getByLabel("Content")).toBeVisible();
      await expect(page.getByLabel("Meta Title")).toBeVisible();
      await expect(page.getByLabel("Meta Description")).toBeVisible();
      await expect(page.getByLabel("Meta Keywords")).toBeVisible();
      await expect(page.getByRole("combobox").first()).toBeVisible();
      await expect(page.getByText("Status").first()).toBeVisible();
      await expect(page.getByLabel(/show in menu|in menu|menu/i)).toBeVisible();
    });

    test("parent page selector lists existing pages", async ({ page }) => {
      await page.goto("/admin/pages");
      await page.locator("main li").first().waitFor();
      await page.getByRole("button", { name: /new page|add page|create page/i }).click();
      await page.getByRole("combobox").first().click();
      const options = page.getByRole("option");
      expect(await options.count()).toBeGreaterThanOrEqual(1);
    });

    test("shows validation errors on empty submission", async ({ page }) => {
      await page.goto("/admin/pages");
      await page.locator("main li").first().waitFor();
      await page.getByRole("button", { name: /new page|add page|create page/i }).click();
      await page.getByRole("button", { name: /save|create|submit/i }).click();
      await expect(page.getByText(/required|cannot be empty/i).first()).toBeVisible();
    });

    test("successful submission adds page to the tree", async ({ page }) => {
      await page.goto("/admin/pages");
      await page.locator("main li").first().waitFor();
      await page.getByRole("button", { name: /new page|add page|create page/i }).click();
      await page.getByLabel(/^Title\b/i).fill("About Us");
      await page.getByLabel("Slug").fill("about");
      await page.getByLabel("Content").fill("About page content");
      await page.getByRole("button", { name: /save|create|submit/i }).click();
      await expect(page.locator("main ul")).toContainText(/about us/i);
    });

    test("cancel closes the form without adding a page", async ({ page }) => {
      await page.goto("/admin/pages");
      await page.locator("main li").first().waitFor();
      await page.getByRole("button", { name: /new page|add page|create page/i }).click();
      await page.getByRole("button", { name: /cancel/i }).click();
      await expect(page.getByRole("dialog")).not.toBeVisible();
    });
  });

  test.describe("Page edit", () => {
    test("edit button opens form pre-filled with page data", async ({ page }) => {
      await page.goto("/admin/pages");
      const firstItem = page.locator("main li").first();
      const title = await firstItem.textContent();
      await firstItem.getByRole("button", { name: /edit|pencil/i }).click();
      await expect(page.getByLabel(/^Title\b/i)).toHaveValue(title?.trim() ?? "");
    });

    test("saving changes updates the tree", async ({ page }) => {
      await page.goto("/admin/pages");
      const firstItem = page.locator("main li").first();
      await firstItem.getByRole("button", { name: /edit|pencil/i }).click();
      await page.getByLabel(/^Title\b/i).fill("Updated Title");
      await page.getByRole("button", { name: /save|update/i }).click();
      await expect(page.locator("main ul")).toContainText(/updated title/i);
    });

    test("cancel edit closes form without changes", async ({ page }) => {
      await page.goto("/admin/pages");
      const firstItem = page.locator("main li").first();
      await firstItem.getByRole("button", { name: /edit|pencil/i }).click();
      await page.getByRole("button", { name: /cancel/i }).click();
      await expect(page.getByRole("dialog")).not.toBeVisible();
    });
  });

  test.describe("Page delete", () => {
    test("delete button shows a confirmation dialog", async ({ page }) => {
      await page.goto("/admin/pages");
      const firstItem = page.locator("main li").first();
      await firstItem.getByRole("button", { name: /delete|remove|trash/i }).click();
      await expect(page.getByRole("dialog")).toBeVisible();
    });

    test("confirming delete removes the page from the tree", async ({ page }) => {
      await page.goto("/admin/pages");
      const firstItem = page.locator("main li").first();
      const title = await firstItem.textContent();
      await firstItem.getByRole("button", { name: /delete|remove|trash/i }).click();
      await page.getByRole("button", { name: /confirm|yes|delete/i }).click();
      await expect(page.getByRole("dialog")).not.toBeVisible();
      await expect(page.locator("main ul")).not.toContainText(title?.trim() ?? "");
    });

    test("cancelling delete keeps the page in the tree", async ({ page }) => {
      await page.goto("/admin/pages");
      const firstItem = page.locator("main li").first();
      const title = await firstItem.textContent();
      await firstItem.getByRole("button", { name: /delete|remove|trash/i }).click();
      await page.getByRole("button", { name: /cancel|no/i }).click();
      await expect(page.getByRole("dialog")).not.toBeVisible();
      await expect(page.locator("main ul")).toContainText(title?.trim() ?? "");
    });
  });

  test.describe("Rich text editor", () => {
    test("content editor has formatting toolbar with Bold, Italic, and Insert Image buttons", async ({ page }) => {
      await page.goto("/admin/pages");
      await page.locator("main li").first().waitFor();
      await page.getByRole("button", { name: /new page|add page|create page/i }).click();
      await expect(page.getByRole("dialog")).toBeVisible();
      await expect(page.getByRole("button", { name: /bold|format_bold/i })).toBeVisible();
      await expect(page.getByRole("button", { name: /italic|format_italic/i })).toBeVisible();
      await expect(page.getByRole("button", { name: /insert image|image/i })).toBeVisible();
    });
  });

  test.describe("Page preview", () => {
    test("edit dialog has Preview button", async ({ page }) => {
      await page.goto("/admin/pages");
      const firstItem = page.locator("main li").first();
      await firstItem.getByRole("button", { name: /edit|pencil/i }).click();
      await expect(page.getByRole("dialog")).toBeVisible();
      await expect(page.getByRole("button", { name: /preview/i })).toBeVisible();
    });

    test("Preview opens page content in new tab", async ({ page }) => {
      await page.goto("/admin/pages");
      const firstItem = page.locator("main li").first();
      const title = await firstItem.textContent();
      await firstItem.getByRole("button", { name: /edit|pencil/i }).click();
      await expect(page.getByRole("dialog")).toBeVisible();

      const [newPage] = await Promise.all([
        page.waitForEvent("popup"),
        page.getByRole("button", { name: /preview/i }).click(),
      ]);
      await newPage.waitForLoadState();
      await expect(newPage.getByText(title?.trim() ?? "")).toBeVisible();
      await newPage.close();
    });
  });

  test.describe("Page reordering", () => {
    test("page tree items have drag handles", async ({ page }) => {
      await page.goto("/admin/pages");
      await expect(page.locator("main ul").first()).toBeVisible();
      const dragIcons = page.locator("main [data-testid='DragIndicatorIcon']");
      expect(await dragIcons.count()).toBeGreaterThanOrEqual(1);
    });

    test("reorder API updates sort order", async ({ page }) => {
      const res1 = await page.request.post("/api/pages", {
        data: { title: "Reorder A", slug: "reorder-a", content: "A" },
      });
      const pageA = await res1.json();
      const res2 = await page.request.post("/api/pages", {
        data: { title: "Reorder B", slug: "reorder-b", content: "B" },
      });
      const pageB = await res2.json();

      const reorderRes = await page.request.put("/api/pages/reorder", {
        data: { parentId: null, pageIds: [pageB.id, pageA.id] },
      });
      expect(reorderRes.ok()).toBeTruthy();

      const pagesRes = await page.request.get("/api/pages");
      const allPages: any[] = await pagesRes.json();
      const reordered = allPages
        .filter((p: any) => p.id === pageA.id || p.id === pageB.id)
        .sort((a: any, b: any) => a.sortOrder - b.sortOrder);
      expect(reordered[0].id).toBe(pageB.id);
      expect(reordered[1].id).toBe(pageA.id);
    });
  });
});
