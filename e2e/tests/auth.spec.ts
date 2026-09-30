import { expect, test } from "@playwright/test";

import { login } from "./helpers";

// login ใหม่ในเทสต์นี้เอง (ไม่ใช้ session ที่ auth.setup.ts เก็บไว้) — logout ของเทสต์นี้จะได้ไม่ไปตัด session ที่เทสต์อื่นใช้
test.describe("เข้าสู่ระบบ / ออกจากระบบ", () => {
  test("session ที่ browser อ่านได้มีแค่ access token — id token อยู่ฝั่ง server เท่านั้น", async ({ page }) => {
    await login(page, "customer1");
    const session = await (await page.request.get("/api/auth/session")).json();
    expect(session.accessToken).toBeTruthy();
    expect(session).not.toHaveProperty("idToken");
    expect(session).not.toHaveProperty("refreshToken");
  });

  test("ออกจากระบบแล้ว session ที่ Keycloak จบด้วย — เข้าหน้าที่ต้อง login ต้องเจอฟอร์ม login อีกครั้ง", async ({ page }) => {
    await login(page, "customer1");
    // รอให้หน้าโหลดข้อมูลเสร็จก่อนกด เหมือนคนกดจริง — ถ้ากดตอนหน้ายังยิง /api/auth/session ค้างอยู่
    // คำตอบที่มาถึงทีหลังจะเขียน session cookie กลับมา (next-auth ต่ออายุ cookie ทุกครั้งที่ตอบ)
    await page.waitForLoadState("networkidle");
    await page.goto("/api/auth/logout");
    await page.waitForURL("http://jongyoung.localhost/");

    // ถ้า Keycloak ยังจำ session อยู่ จะเด้งกลับมา /me/bookings เองโดยไม่ถามรหัสผ่าน
    await page.goto("/me/bookings");
    await page.waitForURL(/keycloak\.jongyoung\.localhost/);
    await expect(page.locator("#username")).toBeVisible();
  });
});
