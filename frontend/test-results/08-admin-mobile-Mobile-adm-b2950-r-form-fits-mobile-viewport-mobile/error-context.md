# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: 08-admin-mobile.spec.ts >> Mobile admin responsive layout >> All tests in this file run only on mobile >> New User form fits mobile viewport
- Location: e2e/08-admin-mobile.spec.ts:50:9

# Error details

```
Test timeout of 30000ms exceeded.
```

```
Error: locator.boundingBox: Test timeout of 30000ms exceeded.
Call log:
  - waiting for locator('[role=\'dialog\'], form').first()

```

# Page snapshot

```yaml
- generic [ref=e1]:
  - generic [ref=e2]:
    - banner [ref=e3]:
      - generic [ref=e4]:
        - button "toggle sidebar" [ref=e5] [cursor=pointer]:
          - img [ref=e6]
        - heading "Admin" [level=6] [ref=e8]
    - main [ref=e9]:
      - generic [ref=e10]:
        - generic [ref=e11]:
          - heading "Users" [level=1] [ref=e12]
          - button "New User" [active] [ref=e13] [cursor=pointer]: New User
        - generic [ref=e15]:
          - textbox "Search users..." [ref=e16]
          - group
        - table [ref=e18]:
          - rowgroup [ref=e19]:
            - row "Name Email Role CreatedAt Actions" [ref=e20]:
              - columnheader "Name" [ref=e21]:
                - button "Name" [ref=e22] [cursor=pointer]:
                  - text: Name
                  - img [ref=e23]
              - columnheader "Email" [ref=e25]:
                - button "Email" [ref=e26] [cursor=pointer]:
                  - text: Email
                  - img [ref=e27]
              - columnheader "Role" [ref=e29]:
                - button "Role" [ref=e30] [cursor=pointer]:
                  - text: Role
                  - img [ref=e31]
              - columnheader "CreatedAt" [ref=e33]:
                - button "CreatedAt" [ref=e34] [cursor=pointer]:
                  - text: CreatedAt
                  - img [ref=e35]
              - columnheader "Actions" [ref=e37]
          - rowgroup [ref=e38]:
            - row "Test User test@example.com user 6/15/2026 Edit Delete" [ref=e39]:
              - cell "Test User" [ref=e40]
              - cell "test@example.com" [ref=e41]
              - cell "user" [ref=e42]
              - cell "6/15/2026" [ref=e43]
              - cell "Edit Delete" [ref=e44]:
                - button "Edit" [ref=e45] [cursor=pointer]
                - button "Delete" [ref=e46] [cursor=pointer]
            - row "Updated Name admin@example.com admin 6/15/2026 Edit Delete" [ref=e47]:
              - cell "Updated Name" [ref=e48]
              - cell "admin@example.com" [ref=e49]
              - cell "admin" [ref=e50]
              - cell "6/15/2026" [ref=e51]
              - cell "Edit Delete" [ref=e52]:
                - button "Edit" [ref=e53] [cursor=pointer]
                - button "Delete" [ref=e54] [cursor=pointer]
  - button "Open Next.js Dev Tools" [ref=e60] [cursor=pointer]:
    - img [ref=e61]
  - alert [ref=e64]
```

# Test source

```ts
  1  | import { test, expect } from "@playwright/test";
  2  | import { isMobile, loginAsAdmin } from "./helpers";
  3  | 
  4  | test.describe("Mobile admin responsive layout", () => {
  5  |   test.beforeEach(async ({ page }) => {
  6  |     await loginAsAdmin(page);
  7  |   });
  8  | 
  9  |   test.describe("All tests in this file run only on mobile", () => {
  10 |     test.beforeAll(() => {
  11 |       test.skip(!isMobile(), "Mobile-specific tests");
  12 |     });
  13 | 
  14 |     test("hamburger menu button is visible", async ({ page }) => {
  15 |       await page.goto("/admin");
  16 |       const hamburger = page.getByRole("button", { name: /menu|hamburger|toggle sidebar/i });
  17 |       await expect(hamburger).toBeVisible();
  18 |     });
  19 | 
  20 |     test("hamburger menu opens sidebar overlay", async ({ page }) => {
  21 |       await page.goto("/admin");
  22 |       const hamburger = page.getByRole("button", { name: /menu|hamburger|toggle sidebar/i });
  23 |       await hamburger.click();
  24 |       const sidebar = page.locator("nav, aside, [role='navigation']").first();
  25 |       await expect(sidebar).toBeVisible();
  26 |     });
  27 | 
  28 |     test("sidebar overlay closes after navigating to a page", async ({ page }) => {
  29 |       await page.goto("/admin");
  30 |       await page.getByRole("button", { name: /menu|hamburger|toggle sidebar/i }).click();
  31 |       await page.getByRole("link", { name: /users/i }).click();
  32 |       await expect(page.locator("nav, aside, [role='navigation']").first()).not.toBeVisible();
  33 |     });
  34 | 
  35 |     test("admin page has no horizontal scroll", async ({ page }) => {
  36 |       await page.goto("/admin");
  37 |       const scrollWidth = await page.evaluate(() => document.documentElement.scrollWidth);
  38 |       const clientWidth = await page.evaluate(() => document.documentElement.clientWidth);
  39 |       expect(scrollWidth).toBeLessThanOrEqual(clientWidth + 1);
  40 |     });
  41 | 
  42 |     test("user table renders as cards on mobile", async ({ page }) => {
  43 |       await page.goto("/admin/users");
  44 |       await page.waitForSelector("table, [role='grid'], [role='list']");
  45 |       const isTable = await page.locator("table, [role='grid']").isVisible();
  46 |       const isCardList = await page.locator("[role='list'] > [role='listitem'], .MuiCard-root").first().isVisible().catch(() => false);
  47 |       expect(isTable || isCardList).toBeTruthy();
  48 |     });
  49 | 
  50 |     test("New User form fits mobile viewport", async ({ page }) => {
  51 |       await page.goto("/admin/users");
  52 |       await page.getByRole("button", { name: /new user|add user|create user/i }).click();
  53 |       const dialog = page.locator("[role='dialog'], form").first();
> 54 |       const box = await dialog.boundingBox();
     |                                ^ Error: locator.boundingBox: Test timeout of 30000ms exceeded.
  55 |       expect(box).not.toBeNull();
  56 |       if (box) {
  57 |         expect(box.width).toBeLessThanOrEqual(400);
  58 |       }
  59 |     });
  60 | 
  61 |     test("form fields have adequate tap targets", async ({ page }) => {
  62 |       await page.goto("/admin/users");
  63 |       await page.getByRole("button", { name: /new user|add user|create user/i }).click();
  64 |       const inputs = page.locator("input, select, button");
  65 |       const count = await inputs.count();
  66 |       for (let i = 0; i < Math.min(count, 5); i++) {
  67 |         const box = await inputs.nth(i).boundingBox();
  68 |         if (box) {
  69 |           expect(box.height).toBeGreaterThanOrEqual(40);
  70 |         }
  71 |       }
  72 |     });
  73 | 
  74 |     test("filter input works on mobile", async ({ page }) => {
  75 |       await page.goto("/admin/users");
  76 |       const filter = page.getByPlaceholder(/filter|search/i);
  77 |       await expect(filter).toBeVisible();
  78 |       await filter.fill("test@example.com");
  79 |       await expect(filter).toHaveValue("test@example.com");
  80 |     });
  81 |   });
  82 | });
  83 | 
```