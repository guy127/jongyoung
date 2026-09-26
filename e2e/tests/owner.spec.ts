import { expect, test } from "@playwright/test";

import { AUTH, restaurantId, tomorrow } from "./helpers";

test.use({ storageState: AUTH.owner });

// เจ้าของร้าน (owner2 — เจ้าของครัวบ้านสวน, ซูชิ, ซีฟู้ด)
test.describe("โหมดเจ้าของร้าน", () => {
  test("บอร์ดรายวันแสดง peak ของรอบ + แถบที่นั่ง + รายการจอง", async ({ page, request }) => {
    // seed: ครัวบ้านสวน (8 ที่) เต็มทั้งรอบพรุ่งนี้ด้วยการจอง 3 รายการ
    const baansuan = await restaurantId(request, "ครัวบ้านสวน");
    await page.goto(`/owner/bookings?restaurant=${baansuan}&date=${tomorrow()}`);
    await expect(page.getByText("โหมดเจ้าของร้าน")).toBeVisible();
    await expect(page.getByLabel("สรุป")).toContainText("8/8 · 11:00");
    await expect(page.getByRole("table").getByRole("row")).toHaveCount(4); // หัวตาราง + 3 รายการ
  });

  test("ลดที่นั่งต่ำกว่าคนที่จองไว้แล้ว → ถูกปฏิเสธพร้อมรหัส SEATS_BELOW_EXISTING_BOOKINGS", async ({ page }) => {
    await page.goto("/owner/restaurants");
    await page.getByRole("row", { name: /ครัวบ้านสวน/ }).getByRole("button", { name: "แก้ไข" }).click();
    await page.getByLabel("จำนวนที่นั่ง *").fill("4");
    await page.getByRole("button", { name: "บันทึก" }).click();
    // กรองด้วยข้อความ: Next.js มี role="alert" ของตัวเอง (route announcer) อยู่ในหน้าด้วย
    const error = page.getByRole("alert").filter({ hasText: "SEATS_BELOW_EXISTING_BOOKINGS" });
    await expect(error).toContainText("ลดที่นั่งไม่ได้");
  });

  test("เปิดหน้าร้านของตัวเอง → ไม่มีฟอร์มรีวิว แต่บอกเหตุผล", async ({ page, request }) => {
    const baansuan = await restaurantId(request, "ครัวบ้านสวน");
    await page.goto(`/restaurants/${baansuan}`);
    await expect(page.getByText("นี่คือร้านของคุณ")).toBeVisible();
    await expect(page.getByText("เขียนรีวิว")).toHaveCount(0);
  });
});
