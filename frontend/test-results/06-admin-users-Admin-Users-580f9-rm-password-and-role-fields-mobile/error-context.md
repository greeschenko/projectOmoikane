# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: 06-admin-users.spec.ts >> Admin Users >> User creation form >> form has name, email, password, confirm password, and role fields
- Location: e2e/06-admin-users.spec.ts:49:9

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByLabel(/^Name\b/i)
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for getByLabel(/^Name\b/i)

```

```yaml
- banner:
  - button "toggle sidebar"
  - heading "Admin" [level=6]
- main:
  - heading "Users" [level=1]
  - button "New User"
  - textbox "Search users..."
  - table:
    - rowgroup:
      - row "Name Email Role CreatedAt Actions":
        - columnheader "Name":
          - button "Name"
        - columnheader "Email":
          - button "Email"
        - columnheader "Role":
          - button "Role"
        - columnheader "CreatedAt":
          - button "CreatedAt"
        - columnheader "Actions"
    - rowgroup:
      - row "Admin admin@example.com admin 6/15/2026 Edit Delete":
        - cell "Admin"
        - cell "admin@example.com"
        - cell "admin"
        - cell "6/15/2026"
        - cell "Edit Delete":
          - button "Edit"
          - button "Delete"
      - row "Test User test@example.com user 6/15/2026 Edit Delete":
        - cell "Test User"
        - cell "test@example.com"
        - cell "user"
        - cell "6/15/2026"
        - cell "Edit Delete":
          - button "Edit"
          - button "Delete"
- alert
```

# Test source

```ts
  1   | import { test, expect } from "@playwright/test";
  2   | import { isMobile, loginAsAdmin } from "./helpers";
  3   | 
  4   | const userFormFieldLabels = [/^Name\b/i, /^Email\b/i, /^Password\b/i, /^Confirm Password\b/i];
  5   | const userRoleOptions = ["admin", "user"];
  6   | 
  7   | test.describe("Admin Users", () => {
  8   |   test.beforeEach(async ({ page }) => {
  9   |     await loginAsAdmin(page);
  10  |   });
  11  | 
  12  |   test.describe("User table", () => {
  13  |     test("has sortable columns", async ({ page }) => {
  14  |       test.skip(isMobile(), "Table sorting switches to dropdown on mobile");
  15  |       await page.goto("/admin/users");
  16  |       await expect(page.getByRole("table")).toBeVisible();
  17  |       const headers = page.getByRole("columnheader");
  18  |       await expect(headers.first()).toBeVisible();
  19  |       const headerTexts = await headers.allTextContents();
  20  |       expect(headerTexts.length).toBeGreaterThan(0);
  21  |     });
  22  | 
  23  |     test("has a filter input", async ({ page }) => {
  24  |       await page.goto("/admin/users");
  25  |       await expect(page.getByPlaceholder(/filter|search/i)).toBeVisible();
  26  |     });
  27  | 
  28  |     test("shows user data in rows", async ({ page }) => {
  29  |       await page.goto("/admin/users");
  30  |       const rows = page.locator("table tbody tr");
  31  |       await expect(rows.first()).toBeVisible();
  32  |     });
  33  | 
  34  |     test("each user row has edit and delete actions", async ({ page }) => {
  35  |       await page.goto("/admin/users");
  36  |       const firstRow = page.locator("table tbody tr").first();
  37  |       await expect(firstRow.getByRole("button", { name: /edit|pencil/i })).toBeVisible();
  38  |       await expect(firstRow.getByRole("button", { name: /delete|remove|trash/i })).toBeVisible();
  39  |     });
  40  |   });
  41  | 
  42  |   test.describe("User creation form", () => {
  43  |     test("'New User' button opens a form", async ({ page }) => {
  44  |       await page.goto("/admin/users");
  45  |       await page.getByRole("button", { name: /new user|add user|create user/i }).click();
  46  |       await expect(page.getByRole("dialog")).toBeVisible();
  47  |     });
  48  | 
  49  |     test("form has name, email, password, confirm password, and role fields", async ({ page }) => {
  50  |       await page.goto("/admin/users");
  51  |       await page.getByRole("button", { name: /new user|add user|create user/i }).click();
  52  |       for (const field of userFormFieldLabels) {
> 53  |         await expect(page.getByLabel(field)).toBeVisible();
      |                                              ^ Error: expect(locator).toBeVisible() failed
  54  |       }
  55  |       await expect(page.getByRole("combobox")).toBeVisible();
  56  |     });
  57  | 
  58  |     test("role selector has admin and user options", async ({ page }) => {
  59  |       await page.goto("/admin/users");
  60  |       await page.getByRole("button", { name: /new user|add user|create user/i }).click();
  61  |       await page.getByRole("combobox").click();
  62  |       for (const option of userRoleOptions) {
  63  |         await expect(page.getByRole("option", { name: new RegExp(option, "i") })).toBeVisible();
  64  |       }
  65  |     });
  66  | 
  67  |     test("shows validation errors on empty submission", async ({ page }) => {
  68  |       await page.goto("/admin/users");
  69  |       await page.getByRole("button", { name: /new user|add user|create user/i }).click();
  70  |       await page.getByRole("button", { name: /save|create|submit/i }).click();
  71  |       await expect(page.getByText(/required|cannot be empty/i).first()).toBeVisible();
  72  |     });
  73  | 
  74  |     test("shows error for invalid email", async ({ page }) => {
  75  |       await page.goto("/admin/users");
  76  |       await page.getByRole("button", { name: /new user|add user|create user/i }).click();
  77  |       await page.getByLabel("Name").fill("New User");
  78  |       await page.getByLabel("Email").fill("not-an-email");
  79  |       await page.getByLabel(/^Password\b/).fill("Password123!");
  80  |       await page.getByLabel("Confirm Password").fill("Password123!");
  81  |       await page.getByRole("button", { name: /save|create|submit/i }).click();
  82  |       await expect(page.getByText(/valid email/i)).toBeVisible();
  83  |     });
  84  | 
  85  |     test("shows error when passwords do not match", async ({ page }) => {
  86  |       await page.goto("/admin/users");
  87  |       await page.getByRole("button", { name: /new user|add user|create user/i }).click();
  88  |       await page.getByLabel("Name").fill("New User");
  89  |       await page.getByLabel("Email").fill("user@example.com");
  90  |       await page.getByLabel(/^Password\b/).fill("Password123!");
  91  |       await page.getByLabel("Confirm Password").fill("DifferentPass1!");
  92  |       await page.getByRole("button", { name: /save|create|submit/i }).click();
  93  |       await expect(page.getByText(/passwords? do not match|passwords? must match/i)).toBeVisible();
  94  |     });
  95  | 
  96  |     test("successful submission adds user to table", async ({ page }) => {
  97  |       await page.goto("/admin/users");
  98  |       await expect(page.locator("table tbody tr")).not.toHaveCount(0);
  99  |       const rowCount = await page.locator("table tbody tr").count();
  100 |       await page.getByRole("button", { name: /new user|add user|create user/i }).click();
  101 |       await page.getByLabel("Name").fill("Jane Doe");
  102 |       await page.getByLabel("Email").fill("jane@example.com");
  103 |       await page.getByLabel(/^Password\b/).fill("SecurePass123!");
  104 |       await page.getByLabel("Confirm Password").fill("SecurePass123!");
  105 |       await page.getByRole("button", { name: /save|create|submit/i }).click();
  106 |       const newRowCount = await page.locator("table tbody tr").count();
  107 |       expect(newRowCount).toBe(rowCount + 1);
  108 |       await expect(page.getByRole("table")).toContainText(/jane@example.com/i);
  109 |     });
  110 | 
  111 |     test("cancel closes the form without adding a user", async ({ page }) => {
  112 |       await page.goto("/admin/users");
  113 |       await expect(page.locator("table tbody tr")).not.toHaveCount(0);
  114 |       const rowCount = await page.locator("table tbody tr").count();
  115 |       await page.getByRole("button", { name: /new user|add user|create user/i }).click();
  116 |       await page.getByRole("button", { name: /cancel/i }).click();
  117 |       await expect(page.locator("[role='dialog'], form")).not.toBeVisible();
  118 |       const afterCancelCount = await page.locator("table tbody tr").count();
  119 |       expect(afterCancelCount).toBe(rowCount);
  120 |     });
  121 |   });
  122 | 
  123 |   test.describe("User edit", () => {
  124 |     test("edit button opens form pre-filled with user data", async ({ page }) => {
  125 |       await page.goto("/admin/users");
  126 |       const firstRow = page.locator("table tbody tr").first();
  127 |       const email = await firstRow.locator("td, [role='gridcell']").nth(1).textContent();
  128 |       await firstRow.getByRole("button", { name: /edit|pencil/i }).click();
  129 |       await expect(page.getByLabel("Email")).toHaveValue(email?.trim() ?? "");
  130 |     });
  131 | 
  132 |     test("saving changes updates the table", async ({ page }) => {
  133 |       await page.goto("/admin/users");
  134 |       const firstRow = page.locator("table tbody tr").first();
  135 |       await firstRow.getByRole("button", { name: /edit|pencil/i }).click();
  136 |       await page.getByLabel("Name").fill("Updated Name");
  137 |       await page.getByRole("button", { name: /save|update/i }).click();
  138 |       await expect(page.getByRole("table")).toContainText(/updated name/i);
  139 |     });
  140 | 
  141 |     test("cancel edit closes form without changes", async ({ page }) => {
  142 |       await page.goto("/admin/users");
  143 |       const firstRow = page.locator("table tbody tr").first();
  144 |       await firstRow.getByRole("button", { name: /edit|pencil/i }).click();
  145 |       await page.getByRole("button", { name: /cancel/i }).click();
  146 |       await expect(page.locator("[role='dialog'], form")).not.toBeVisible();
  147 |     });
  148 |   });
  149 | 
  150 |   test.describe("User delete", () => {
  151 |     test("delete button shows a confirmation dialog", async ({ page }) => {
  152 |       await page.goto("/admin/users");
  153 |       const firstRow = page.locator("table tbody tr").first();
```