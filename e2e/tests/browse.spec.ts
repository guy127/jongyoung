import { expect, test } from "@playwright/test";

import { nextMonday, tomorrow } from "./helpers";

// ผู้ใช้ที่ยังไม่ login: ค้นหา → ดูเวลาว่างบนการ์ด → กดเวลาไปหน้าร้าน (รันทั้ง desktop และมือถือ 375px)
test.describe("ค้นหาร้าน (ไม่ต้อง login)", () => {
  test("หน้าแรกแสดงร้านทั้งหมด พร้อมปุ่มเวลา 5 ช่วงรอบเวลาที่ค้น", async ({ page }) => {
    await page.goto(`/?date=${tomorrow()}&time=19:00&party_size=2`);
    await expect(page.locator("article")).toHaveCount(6);
    const chips = page.getByRole("group", { name: "เวลาว่างของ ซูชิ ทาคุมิ" }).getByRole("link");
    await expect(chips).toHaveCount(5);
    await expect(chips.nth(2)).toHaveAccessibleName(/^19:00/);
  });

  test("เรียงคะแนนสูงสุดแบบ Bayesian: ★5.0 จาก 1 รีวิวไม่แซง ★4.8 จาก 46 รีวิว และร้านไม่มีรีวิวอยู่ท้ายสุด", async ({ page }) => {
    await page.goto(`/?sort=rating&date=${tomorrow()}&time=19:00`);
    const names = await page.locator("article .card-title").allTextContents();
    expect(names[0]).toBe("ซูชิ ทาคุมิ");
    expect(names.indexOf("บ้านชาบู บุฟเฟ่ต์")).toBeGreaterThan(names.indexOf("ซูชิ ทาคุมิ"));
    expect(names.at(-1)).toBe("โจ๊กสามย่าน 24 ชม.");
    // ร้านรีวิวน้อยแสดงป้าย "รีวิวน้อย" แทนดาวเด่น
    await expect(page.locator("article", { hasText: "บ้านชาบู" }).getByText("รีวิวน้อย (1)")).toBeVisible();
  });

  test("ร้านที่เต็มทั้งรอบไม่ถูกซ่อน แต่เสนอวันถัดไปที่ว่าง", async ({ page }) => {
    await page.goto(`/?date=${tomorrow()}&time=19:00&party_size=2`);
    const card = page.locator("article", { hasText: "ครัวบ้านสวน" });
    await expect(card.getByText("เต็มช่วงเวลานี้")).toBeVisible();
    await expect(card.getByText(/ว่างวันถัดไป/)).toBeVisible();
  });

  test("ร้านข้ามเที่ยงคืน: เวลาหลังเที่ยงคืนมีป้ายบอกวัน และกดแล้วไปหน้าร้านพร้อมเวลาที่เลือก", async ({ page }) => {
    await page.goto(`/?date=${tomorrow()}&time=23:30&party_size=2`);
    const chips = page.getByRole("group", { name: "เวลาว่างของ ท่าเรือซีฟู้ดบาร์" });
    await expect(chips.getByRole("link", { name: /^00:30 \(เช้าวันที่ \d+\)/ })).toBeVisible();

    await chips.getByRole("link", { name: /^23:30/ }).click();
    await expect(page.getByRole("heading", { level: 1, name: "ท่าเรือซีฟู้ดบาร์" })).toBeVisible();
    await expect(page.getByRole("button", { name: /^23:30/, pressed: true })).toBeVisible();
  });

  test("หน้าที่ต้อง login พาไปหน้า login ของ Keycloak (ธีมจองยัง ภาษาไทย) โดยไม่มีหน้าคั่นกลาง", async ({ page }) => {
    await page.goto("/me/bookings");
    await page.waitForURL(/keycloak\.jongyoung\.localhost/);
    await expect(page.getByRole("heading", { name: "เข้าสู่ระบบเพื่อจองโต๊ะ" })).toBeVisible();
  });

  test("วันปิดประจำสัปดาห์: การ์ดและแผงจองบอกว่าร้านปิด ไม่ใช่ร้านเต็ม", async ({ page }) => {
    // seed: บ้านชาบู บุฟเฟ่ต์ ปิดทุกวันจันทร์
    await page.goto(`/?date=${nextMonday()}&time=19:00&party_size=2`);
    const card = page.locator("article", { hasText: "บ้านชาบู" });
    await expect(card.getByText("ร้านปิดวันนี้ (ปิดทุกวันจันทร์)")).toBeVisible();
    await expect(card.getByText("เต็มช่วงเวลานี้")).toHaveCount(0);

    await card.getByRole("link", { name: "บ้านชาบู บุฟเฟ่ต์", exact: true }).click();
    await expect(page.getByText("วันนี้ร้านปิด (ปิดทุกวันจันทร์) ลองเลือกวันอื่น")).toBeVisible();
  });
});
