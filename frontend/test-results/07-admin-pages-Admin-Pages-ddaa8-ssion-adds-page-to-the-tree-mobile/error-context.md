# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: 07-admin-pages.spec.ts >> Admin Pages >> Page creation form >> successful submission adds page to the tree
- Location: e2e/07-admin-pages.spec.ts:67:9

# Error details

```
Test timeout of 30000ms exceeded.
```

```
Error: locator.fill: Test timeout of 30000ms exceeded.
Call log:
  - waiting for getByLabel(/^Title\b/i)

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
          - heading "Pages" [level=1] [ref=e12]
          - button "New Page" [active] [ref=e13] [cursor=pointer]: New Page
        - list [ref=e15]:
          - listitem [ref=e16]:
            - generic [ref=e17]:
              - paragraph [ref=e18]: Test Page 1
              - button "edit" [ref=e19] [cursor=pointer]:
                - img [ref=e20]
              - button "delete" [ref=e22] [cursor=pointer]:
                - img [ref=e23]
          - listitem [ref=e25]:
            - generic [ref=e26]:
              - paragraph [ref=e27]: Test Page 2
              - button "edit" [ref=e28] [cursor=pointer]:
                - img [ref=e29]
              - button "delete" [ref=e31] [cursor=pointer]:
                - img [ref=e32]
  - button "Open Next.js Dev Tools" [ref=e39] [cursor=pointer]:
    - img [ref=e40]
  - alert [ref=e43]
```

# Test source

```ts
  1   | import { test, expect } from "@playwright/test";
  2   | import { isMobile, loginAsAdmin } from "./helpers";
  3   | 
  4   | test.describe("Admin Pages", () => {
  5   |   test.beforeEach(async ({ page }) => {
  6   |     await loginAsAdmin(page);
  7   |   });
  8   | 
  9   |   test.describe("Page tree", () => {
  10  |     test("shows page tree with parent-child structure", async ({ page }) => {
  11  |       await page.goto("/admin/pages");
  12  |       const tree = page.locator("main ul").first();
  13  |       await expect(tree).toBeVisible();
  14  |     });
  15  | 
  16  |     test("has 'New Page' button", async ({ page }) => {
  17  |       await page.goto("/admin/pages");
  18  |       await expect(page.getByRole("button", { name: /new page|add page|create page/i })).toBeVisible();
  19  |     });
  20  | 
  21  |     test("each page has edit and delete actions", async ({ page }) => {
  22  |       test.skip(isMobile(), "Action buttons may be in a swipe menu on mobile");
  23  |       await page.goto("/admin/pages");
  24  |       const firstItem = page.locator("main li").first();
  25  |       await expect(firstItem.getByRole("button", { name: /edit|pencil/i })).toBeVisible();
  26  |       await expect(firstItem.getByRole("button", { name: /delete|remove|trash/i })).toBeVisible();
  27  |     });
  28  |   });
  29  | 
  30  |   test.describe("Page creation form", () => {
  31  |     const formFields = ["Title", "Slug", "Content"];
  32  | 
  33  |     test("'New Page' button opens a form", async ({ page }) => {
  34  |       await page.goto("/admin/pages");
  35  |       await page.getByRole("button", { name: /new page|add page|create page/i }).click();
  36  |       await expect(page.getByRole("dialog")).toBeVisible();
  37  |     });
  38  | 
  39  |     test("form has title, slug, content, meta fields, parent selector, and published toggle", async ({ page }) => {
  40  |       await page.goto("/admin/pages");
  41  |       await page.getByRole("button", { name: /new page|add page|create page/i }).click();
  42  |       await expect(page.getByLabel(/^Title\b/i)).toBeVisible();
  43  |       await expect(page.getByLabel("Slug")).toBeVisible();
  44  |       await expect(page.getByLabel("Content")).toBeVisible();
  45  |       await expect(page.getByLabel("Meta Title")).toBeVisible();
  46  |       await expect(page.getByLabel("Meta Description")).toBeVisible();
  47  |       await expect(page.getByLabel("Meta Keywords")).toBeVisible();
  48  |       await expect(page.getByRole("combobox")).toBeVisible();
  49  |       await expect(page.getByLabel("Published")).toBeVisible();
  50  |     });
  51  | 
  52  |     test("parent page selector lists existing pages", async ({ page }) => {
  53  |       await page.goto("/admin/pages");
  54  |       await page.getByRole("button", { name: /new page|add page|create page/i }).click();
  55  |       await page.getByRole("combobox").click();
  56  |       const options = page.getByRole("option");
  57  |       expect(await options.count()).toBeGreaterThanOrEqual(1);
  58  |     });
  59  | 
  60  |     test("shows validation errors on empty submission", async ({ page }) => {
  61  |       await page.goto("/admin/pages");
  62  |       await page.getByRole("button", { name: /new page|add page|create page/i }).click();
  63  |       await page.getByRole("button", { name: /save|create|submit/i }).click();
  64  |       await expect(page.getByText(/required|cannot be empty/i).first()).toBeVisible();
  65  |     });
  66  | 
  67  |     test("successful submission adds page to the tree", async ({ page }) => {
  68  |       await page.goto("/admin/pages");
  69  |       await page.getByRole("button", { name: /new page|add page|create page/i }).click();
> 70  |       await page.getByLabel(/^Title\b/i).fill("About Us");
      |                                          ^ Error: locator.fill: Test timeout of 30000ms exceeded.
  71  |       await page.getByLabel("Slug").fill("about");
  72  |       await page.getByLabel("Content").fill("About page content");
  73  |       await page.getByRole("button", { name: /save|create|submit/i }).click();
  74  |       await expect(page.locator("main ul")).toContainText(/about us/i);
  75  |     });
  76  | 
  77  |     test("cancel closes the form without adding a page", async ({ page }) => {
  78  |       await page.goto("/admin/pages");
  79  |       await page.getByRole("button", { name: /new page|add page|create page/i }).click();
  80  |       await page.getByRole("button", { name: /cancel/i }).click();
  81  |       await expect(page.getByRole("dialog")).not.toBeVisible();
  82  |     });
  83  |   });
  84  | 
  85  |   test.describe("Page edit", () => {
  86  |     test("edit button opens form pre-filled with page data", async ({ page }) => {
  87  |       await page.goto("/admin/pages");
  88  |       const firstItem = page.locator("main li").first();
  89  |       const title = await firstItem.textContent();
  90  |       await firstItem.getByRole("button", { name: /edit|pencil/i }).click();
  91  |       await expect(page.getByLabel(/^Title\b/i)).toHaveValue(title?.trim() ?? "");
  92  |     });
  93  | 
  94  |     test("saving changes updates the tree", async ({ page }) => {
  95  |       await page.goto("/admin/pages");
  96  |       const firstItem = page.locator("main li").first();
  97  |       await firstItem.getByRole("button", { name: /edit|pencil/i }).click();
  98  |       await page.getByLabel(/^Title\b/i).fill("Updated Title");
  99  |       await page.getByRole("button", { name: /save|update/i }).click();
  100 |       await expect(page.locator("main ul")).toContainText(/updated title/i);
  101 |     });
  102 | 
  103 |     test("cancel edit closes form without changes", async ({ page }) => {
  104 |       await page.goto("/admin/pages");
  105 |       const firstItem = page.locator("main li").first();
  106 |       await firstItem.getByRole("button", { name: /edit|pencil/i }).click();
  107 |       await page.getByRole("button", { name: /cancel/i }).click();
  108 |       await expect(page.getByRole("dialog")).not.toBeVisible();
  109 |     });
  110 |   });
  111 | 
  112 |   test.describe("Page delete", () => {
  113 |     test("delete button shows a confirmation dialog", async ({ page }) => {
  114 |       await page.goto("/admin/pages");
  115 |       const firstItem = page.locator("main li").first();
  116 |       await firstItem.getByRole("button", { name: /delete|remove|trash/i }).click();
  117 |       await expect(page.getByRole("dialog")).toBeVisible();
  118 |     });
  119 | 
  120 |     test("confirming delete removes the page from the tree", async ({ page }) => {
  121 |       await page.goto("/admin/pages");
  122 |       const firstItem = page.locator("main li").first();
  123 |       const title = await firstItem.textContent();
  124 |       await firstItem.getByRole("button", { name: /delete|remove|trash/i }).click();
  125 |       await page.getByRole("button", { name: /confirm|yes|delete/i }).click();
  126 |       await expect(page.getByRole("dialog")).not.toBeVisible();
  127 |       await expect(page.locator("main ul")).not.toContainText(title?.trim() ?? "");
  128 |     });
  129 | 
  130 |     test("cancelling delete keeps the page in the tree", async ({ page }) => {
  131 |       await page.goto("/admin/pages");
  132 |       const firstItem = page.locator("main li").first();
  133 |       const title = await firstItem.textContent();
  134 |       await firstItem.getByRole("button", { name: /delete|remove|trash/i }).click();
  135 |       await page.getByRole("button", { name: /cancel|no/i }).click();
  136 |       await expect(page.getByRole("dialog")).not.toBeVisible();
  137 |       await expect(page.locator("main ul")).toContainText(title?.trim() ?? "");
  138 |     });
  139 |   });
  140 | });
  141 | 
```