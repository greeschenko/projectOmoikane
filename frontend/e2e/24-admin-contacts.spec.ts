import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";

test.describe("Admin Contacts", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("contacts page shows empty state", async ({ page }) => {
    await page.goto("/admin/contacts");
    await expect(page.getByText(/no contact messages/i)).toBeVisible();
  });

  test("delete button shows confirmation dialog", async ({ page }) => {
    // Seed a contact via API
    await page.request.post("/api/contact", {
      data: { name: "Test User", email: "test@test.com", subject: "Test Subject", message: "Test message body" },
    });
    await page.goto("/admin/contacts");
    await expect(page.getByText("Test Subject")).toBeVisible();
    await page.getByRole("button", { name: /delete/i }).first().click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await expect(page.getByText(/delete this message|confirm/i)).toBeVisible();
  });

  test("confirming delete removes the contact", async ({ page }) => {
    await page.request.post("/api/contact", {
      data: { name: "Delete Me", email: "delete@test.com", subject: "Delete Subject", message: "Delete this" },
    });
    await page.goto("/admin/contacts");
    await expect(page.getByText("Delete Subject")).toBeVisible();
    await page.getByRole("button", { name: /delete/i }).first().click();
    await page.getByRole("button", { name: /confirm|yes|delete/i }).first().click();
    await expect(page.getByRole("dialog")).not.toBeVisible();
    await expect(page.getByText("Delete Subject")).toHaveCount(0);
  });

  test("cancelling delete keeps the contact", async ({ page }) => {
    await page.request.post("/api/contact", {
      data: { name: "Keep Me", email: "keep@test.com", subject: "Keep Subject", message: "Keep this" },
    });
    await page.goto("/admin/contacts");
    await expect(page.getByText("Keep Subject")).toBeVisible();
    await page.getByRole("button", { name: /delete/i }).first().click();
    await page.getByRole("button", { name: /cancel/i }).click();
    await expect(page.getByRole("dialog")).not.toBeVisible();
    await expect(page.getByText("Keep Subject")).toBeVisible();
  });
});
