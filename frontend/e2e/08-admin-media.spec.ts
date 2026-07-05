import { test, expect } from "@playwright/test";
import { loginAsAdmin } from "./helpers";
import path from "path";

test.describe("Admin Media Library", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("media page shows empty state with upload button", async ({ page }) => {
    await page.goto("/admin/media");
    await expect(page.getByRole("heading", { name: /media library/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /upload/i })).toBeVisible();
    await expect(page.getByText(/no media uploaded/i)).toBeVisible();
  });

  test("upload dialog opens on clicking upload button", async ({ page }) => {
    const mediaResponse = page.waitForResponse(r => r.url().includes('/api/media') && r.status() === 200);
    await page.goto("/admin/media");
    await mediaResponse;
    await page.getByRole("button", { name: /^upload$/i }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await expect(page.getByRole("heading", { name: /upload media/i })).toBeVisible();
  });

  test("upload image and display in gallery", async ({ page }) => {
    const mediaResponse = page.waitForResponse(r => r.url().includes('/api/media') && r.status() === 200);
    await page.goto("/admin/media");
    await mediaResponse;
    await page.getByRole("button", { name: /^upload$/i }).click();
    await page.locator('input[type="file"]').setInputFiles({
      name: "test.png",
      mimeType: "image/png",
      buffer: Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==", "base64"),
    });
    await page.getByRole("button", { name: /^upload$/i }).last().click();
    await expect(page.getByText("test.png")).toBeVisible();
  });

  test("delete media removes it from gallery", async ({ page }) => {
    const mediaResponse = page.waitForResponse(r => r.url().includes('/api/media') && r.status() === 200);
    await page.goto("/admin/media");
    await mediaResponse;
    await page.getByRole("button", { name: /^upload$/i }).click();
    await page.locator('input[type="file"]').setInputFiles({
      name: "delete-me.png",
      mimeType: "image/png",
      buffer: Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==", "base64"),
    });
    await page.getByRole("button", { name: /^upload$/i }).last().click();
    await expect(page.getByText("delete-me.png")).toBeVisible();
    await page.getByRole("button", { name: /delete/i }).first().click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await page.getByRole("button", { name: /^delete$/i }).last().click();
    await expect(page.getByRole("dialog")).not.toBeVisible({ timeout: 5000 });
    await expect(page.getByText("delete-me.png", { exact: true })).toHaveCount(0);
  });
});
