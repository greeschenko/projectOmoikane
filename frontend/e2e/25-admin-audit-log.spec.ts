import { test, expect } from "@playwright/test";
import { isMobile, loginAsAdmin } from "./helpers";

test.describe("Admin Audit Log", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("sidebar has Audit Log nav link", async ({ page }) => {
    test.skip(isMobile(), "Sidebar is collapsed behind hamburger on mobile");
    await expect(page.getByRole("link", { name: /audit log/i })).toBeVisible();
  });

  test("audit log page loads and shows heading", async ({ page }) => {
    await page.goto("/admin/audit-logs");
    await expect(page.getByRole("heading", { name: /audit log/i })).toBeVisible();
  });

  test("audit log page shows table with entries count", async ({ page }) => {
    await page.goto("/admin/audit-logs");
    await expect(page.getByText(/entries/)).toBeVisible();
  });

  test("audit log page has entity filter tabs", async ({ page }) => {
    await page.goto("/admin/audit-logs");
    await expect(page.getByRole("tab", { name: /all/i })).toBeVisible();
    await expect(page.getByRole("tab", { name: /user/i })).toBeVisible();
    await expect(page.getByRole("tab", { name: /page/i })).toBeVisible();
  });

  test("audit log page has search input", async ({ page }) => {
    await page.goto("/admin/audit-logs");
    await expect(page.getByPlaceholder(/search/i)).toBeVisible();
  });

  // Phase 32 gate: the audit write path is the Kafka backbone, not HTTP. This
  // proves the full pipeline end to end — content-service publishes
  // post.published via its outbox relay -> Kafka -> audit-service consumer
  // (group "audit") -> audit_logs row -> readable via the admin API.
  test("publishing a blog post produces an audit row via Kafka", async ({ page }) => {
    const unique = `Phase32 Post ${Date.now()}`;
    const res = await page.request.post("/api/blog/posts", {
      data: { title: unique, slug: `phase32-${Date.now()}`, content: "<p>hello</p>", status: "published" },
    });
    expect(res.ok()).toBeTruthy();

    await expect
      .poll(
        async () => {
          const r = await page.request.get("/api/audit-logs", {
            params: { entity: "post", search: unique },
          });
          if (!r.ok()) return "";
          const body = await r.json();
          const hit = (body?.logs ?? []).find(
            (l: { action?: string; detail?: string }) =>
              l.action === "publish" && (l.detail ?? "").includes(unique)
          );
          return hit?.detail ?? "";
        },
        { timeout: 20000, intervals: [1000] }
      )
      .toContain(unique);
  });
});
