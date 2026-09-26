import { expect, test } from "@playwright/test";

import { AUTH, daysFromToday, restaurantId, tomorrow } from "./helpers";

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

  test("ตั้งวันปิดประจำสัปดาห์ทับวันที่มีคนจอง → ถูกปฏิเสธพร้อมรหัส HOURS_CONFLICT_EXISTING_BOOKINGS", async ({ page }) => {
    // seed: ครัวบ้านสวนมีการจองพรุ่งนี้ → ปิดวันในสัปดาห์ของพรุ่งนี้ไม่ได้
    const weekday = ["อาทิตย์", "จันทร์", "อังคาร", "พุธ", "พฤหัสบดี", "ศุกร์", "เสาร์"][new Date(`${tomorrow()}T12:00:00+07:00`).getUTCDay()];
    await page.goto("/owner/restaurants");
    await page.getByRole("row", { name: /ครัวบ้านสวน/ }).getByRole("button", { name: "แก้ไข" }).click();
    await page.getByRole("group", { name: "วันปิดประจำสัปดาห์" }).getByLabel(weekday, { exact: true }).check();
    await page.getByRole("button", { name: "บันทึก" }).click();
    await expect(page.getByRole("alert").filter({ hasText: "HOURS_CONFLICT_EXISTING_BOOKINGS" })).toBeVisible();
  });

  test("ปิดร้านทับการจอง → เห็นรายการ → ยืนยัน → ลูกค้าเห็นแจ้งเตือนและป้ายร้านยกเลิก", async ({ page, browser, request }) => {
    // seed: customer1 จองซูชิ (ร้านของ owner2) ไว้ 3 วันข้างหน้า 19:00–20:30
    const sushi = await restaurantId(request, "ซูชิ");
    await page.goto(`/owner/closures?restaurant=${sushi}&date=${daysFromToday(3)}&start=18:00&end=21:00`);
    await page.getByLabel(/เหตุผล/).fill("ทดสอบปิดร้าน");
    await page.getByRole("button", { name: "บันทึกช่วงปิด" }).click();
    await expect(page.getByRole("heading", { name: /การจองที่จะถูกยกเลิก 1 รายการ/ })).toBeVisible();
    await expect(page.getByRole("table")).toContainText("มะลิ วงศ์ดี");
    await page.getByRole("button", { name: "ยืนยันปิดร้านและยกเลิก 1 การจอง" }).click();
    await expect(page.getByRole("status").filter({ hasText: "ปิดร้านแล้ว" })).toBeVisible();

    const customer = await (await browser.newContext({ storageState: AUTH.customer })).newPage();
    await customer.goto("/");
    await customer.getByRole("button", { name: /การแจ้งเตือน \d+ รายการที่ยังไม่อ่าน/ }).click();
    await customer.getByRole("button", { name: /ซูชิ.*ยกเลิกการจองของคุณ.*ทดสอบปิดร้าน/ }).click();
    await expect(customer).toHaveURL(/\/bookings\/[0-9a-f-]{36}$/);
    await expect(customer.getByText("ร้านยกเลิกการจองนี้")).toBeVisible();
    await expect(customer.getByText("เหตุผลจากร้าน: ทดสอบปิดร้าน")).toBeVisible();
  });
});
