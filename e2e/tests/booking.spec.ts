import { expect, test } from "@playwright/test";

import { AUTH, restaurantId, tomorrow } from "./helpers";

test.use({ storageState: AUTH.customer });

// ลูกค้า (customer1): จอง → หน้ายืนยัน → การจองของฉัน → ยกเลิก และกรณีที่ API ตอบ 409
test.describe("การจอง (ลูกค้า)", () => {
  test("จองสำเร็จ → เห็นหน้ายืนยัน → อยู่ในการจองของฉัน → ยกเลิกแล้วย้ายไปแท็บยกเลิกแล้ว", async ({ page, request }) => {
    const sushi = await restaurantId(request, "ซูชิ");
    await page.goto(`/restaurants/${sushi}?date=${tomorrow()}&time=12:00&party_size=2`);

    // กติกายกเลิกต้องเห็นก่อนกดยืนยัน
    await expect(page.getByText(/ยกเลิกหรือแก้ไขได้ถึง/)).toBeVisible();
    await page.getByRole("button", { name: "ยืนยันการจอง" }).click();

    await expect(page.getByRole("heading", { name: "จองสำเร็จแล้ว" })).toBeVisible();
    const code = (await page.getByText(/^JY-[0-9A-F]{6}$/).textContent())!;
    await expect(page.getByRole("button", { name: /เพิ่มลงปฏิทิน/ })).toBeVisible();

    await page.getByRole("link", { name: "ดูการจองของฉัน" }).click();
    const card = page.locator("article", { hasText: code });
    await expect(card).toContainText("12:00–13:00");
    await expect(card).toContainText("✓ ยืนยันแล้ว");

    await card.getByRole("button", { name: "ยกเลิกการจอง" }).click();
    const dialog = page.getByRole("alertdialog", { name: "ยืนยันการยกเลิก" });
    await expect(dialog).toContainText("ที่นั่ง 2 ที่");
    await dialog.getByRole("button", { name: "ยืนยันยกเลิก" }).click();
    await expect(card).toHaveCount(0);

    await page.getByRole("tab", { name: "ยกเลิกแล้ว" }).click();
    await expect(page.locator("article", { hasText: code })).toContainText("✕ ยกเลิกแล้ว");
  });

  test("จองซ้อนเวลาที่ตัวเองจองไว้ → บอกว่ามีการจองเดิม พร้อมลิงก์ไปดู (ไม่บอกว่าร้านเต็ม)", async ({ page, request }) => {
    // seed: customer1 จองร้านซีฟู้ดไว้แล้ว พรุ่งนี้ 23:30–00:30
    const seafood = await restaurantId(request, "ซีฟู้ด");
    await page.goto(`/restaurants/${seafood}?date=${tomorrow()}&time=23:30&party_size=2`);
    await page.getByRole("button", { name: "ยืนยันการจอง" }).click();

    const alert = page.getByRole("status").filter({ hasText: "คุณมีการจองที่ทับช่วงนี้อยู่แล้ว" });
    await expect(alert).toBeVisible();
    await expect(alert).not.toContainText("เต็ม");
    await alert.getByRole("link", { name: "ดูรายการจองเดิม" }).click();
    await expect(page).toHaveURL(/\/bookings\/[0-9a-f-]{36}$/);
    await expect(page.getByText("23:30–00:30")).toBeVisible();
  });

  test("ช่วงที่ที่นั่งไม่พอสำหรับจำนวนคนที่เลือก → ปุ่มเวลากดไม่ได้จริง (disabled) และบอกว่าเต็ม", async ({ page, request }) => {
    // seed: ร้านตามสั่ง (10 ที่) พรุ่งนี้ 18:30–19:30 มีคนจองแล้ว 8 → เหลือ 2 แต่ขอ 4 คน
    const tamsang = await restaurantId(request, "ตามสั่ง");
    await page.goto(`/restaurants/${tamsang}?date=${tomorrow()}&party_size=4`);
    const full = page.getByRole("group", { name: "เวลาเริ่ม" }).getByRole("button", { name: "18:30 เต็ม" });
    await expect(full).toBeDisabled();
  });
});
