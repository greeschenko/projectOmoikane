import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import { loginAsAdmin } from "./helpers";

type Route = { path: string; name: string };

const PUBLIC_ROUTES: Route[] = [
  { path: "/", name: "home" },
  { path: "/blog", name: "blog list" },
  { path: "/contact", name: "contact" },
  { path: "/login", name: "login" },
];

const ADMIN_ROUTES: Route[] = [
  { path: "/admin", name: "dashboard" },
  { path: "/admin/users", name: "users" },
  { path: "/admin/blog", name: "blog admin" },
  { path: "/admin/media", name: "media" },
  { path: "/admin/settings", name: "settings" },
  { path: "/admin/trash", name: "trash" },
  { path: "/admin/audit-logs", name: "audit logs" },
];

const BLOCKING_IMPACTS = ["critical", "serious"] as const;

async function scanAndAssert(page: import("@playwright/test").Page, label: string) {
  const results = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
    .analyze();
  const blocking = results.violations.filter((v) =>
    BLOCKING_IMPACTS.includes(v.impact as (typeof BLOCKING_IMPACTS)[number]),
  );
  if (blocking.length > 0) {
    console.log(
      `A11Y violations on ${label}:`,
      blocking.map((v) => ({ id: v.id, impact: v.impact, help: v.help, nodes: v.nodes.length })),
    );
  }
  expect(blocking, `No critical/serious a11y violations on ${label}`).toEqual([]);
}

test.describe("Accessibility scan — public routes", () => {
  for (const route of PUBLIC_ROUTES) {
    test(`no critical/serious violations: ${route.name}`, async ({ page }) => {
      await page.goto(route.path, { waitUntil: "domcontentloaded" });
      await scanAndAssert(page, route.path);
    });
  }

  test("no critical/serious violations: page detail", async ({ page }) => {
    // Seed a published page first (loginAsAdmin recreates two test pages)
    await loginAsAdmin(page);
    await page.goto("/pages/testpage1", { waitUntil: "domcontentloaded" });
    await scanAndAssert(page, "/pages/testpage1");
  });
});

test.describe("Accessibility scan — admin routes", () => {
  for (const route of ADMIN_ROUTES) {
    test(`no critical/serious violations: ${route.name}`, async ({ page }) => {
      await loginAsAdmin(page);
      await page.goto(route.path, { waitUntil: "domcontentloaded" });
      await scanAndAssert(page, route.path);
    });
  }
});
