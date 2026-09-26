import { expect, test, type Page } from "@playwright/test";

import { AUTH } from "./helpers";

// ทางเข้าฝั่งเจ้าของร้านจาก navbar ต้องมีเสมอ (ข้อ 5.1: ทุกคนที่ login สร้างร้านได้) — ทั้ง desktop และมือถือ 375px
const nav = (page: Page) => page.getByRole("navigation", { name: "เมนูหลัก" });

// navbar ต้องไม่ล้นจอ (ปุ่มขวาสุดยังอยู่ในจอ)
async function expectNoOverflow(page: Page, name = "เมนูหลัก") {
  const el = page.getByRole("navigation", { name });
  const box = await el.boundingBox();
  expect(box!.x + box!.width).toBeLessThanOrEqual(page.viewportSize()!.width);
  expect(await el.evaluate((n) => n.scrollWidth <= n.clientWidth)).toBeTruthy();
}

for (const viewport of [{ width: 1280, height: 900 }, { width: 375, height: 812 }]) {
  test.describe(`ทางเข้าเจ้าของร้าน @${viewport.width}px`, () => {
    test.use({ viewport });

    test.describe("ลูกค้าที่ยังไม่มีร้าน", () => {
      test.use({ storageState: AUTH.customer });

      test("เห็นลิงก์ เปิดร้านของคุณ → ไปหน้าร้านของฉัน", async ({ page }) => {
        await page.goto("/");
        const open = nav(page).getByRole("link", { name: /เปิดร้าน/ });
        await expect(open).toBeVisible();
        await expectNoOverflow(page);
        await open.click();
        await expect(page).toHaveURL(/\/owner\/restaurants/);
        await expect(page.getByText("ยังไม่มีร้าน")).toBeVisible();
      });
    });

    test.describe("เจ้าของร้าน", () => {
      test.use({ storageState: AUTH.owner });

      test("เห็นตัวสลับโหมด → เข้าโหมดเจ้าของร้าน", async ({ page }) => {
        await page.goto("/");
        const group = nav(page).getByRole("group", { name: "โหมดการใช้งาน" });
        await expect(group).toBeVisible();
        await expect(nav(page).getByRole("link", { name: /เปิดร้าน/ })).toHaveCount(0);
        await expectNoOverflow(page);
        await group.getByRole("link").click();
        await expect(page).toHaveURL(/\/owner\/restaurants/);
        // navbar โหมดเจ้าของร้านก็ต้องใช้ได้ที่มือถือ — ตัวสลับกลับ + เมนูอยู่ในจอครบ
        const ownerNav = page.getByRole("navigation", { name: "เมนูเจ้าของร้าน" });
        await expect(ownerNav.getByRole("link", { name: "การจองรายวัน" })).toBeVisible();
        await expect(ownerNav.getByRole("link", { name: "ลูกค้า" })).toBeVisible();
        await expectNoOverflow(page, "เมนูเจ้าของร้าน");
      });
    });
  });
}
