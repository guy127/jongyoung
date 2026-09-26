import { expect, type APIRequestContext, type Page } from "@playwright/test";

export const API = "http://api.jongyoung.localhost/api/v1";

// ไฟล์เก็บ session ที่ login แล้ว (สร้างใน auth.setup.ts) — อยู่นอกโฟลเดอร์โปรเจกต์ ไม่ต้อง gitignore
export const AUTH = {
  customer: "/tmp/jongyoung-e2e/customer1.json",
  owner: "/tmp/jongyoung-e2e/owner2.json",
};

/** วันพรุ่งนี้ตามเวลาไทย (YYYY-MM-DD) — seed สร้างข้อมูลทดสอบไว้ที่วันนี้ */
export function tomorrow() {
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Bangkok" }).format(new Date(Date.now() + 86_400_000));
}

/** วันจันทร์ถัดไป (หลังวันนี้) ตามเวลาไทย — seed ให้ "บ้านชาบู บุฟเฟ่ต์" ปิดทุกวันจันทร์ */
export function nextMonday() {
  const today = new Date(`${tomorrow()}T12:00:00+07:00`).getTime() - 86_400_000;
  for (let i = 1; i <= 7; i++) {
    const d = new Date(today + i * 86_400_000);
    if (d.getUTCDay() === 1) return new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Bangkok" }).format(d);
  }
  throw new Error("unreachable");
}

/** หา id ร้านจากชื่อ (seed --reset สร้าง id ใหม่ทุกครั้ง จึงอ้างด้วยชื่อ) */
export async function restaurantId(request: APIRequestContext, name: string) {
  const res = await request.get(`${API}/restaurants?limit=50`);
  expect(res.ok()).toBeTruthy();
  const { items } = (await res.json()) as { items: { id: string; name: string }[] };
  const found = items.find((r) => r.name.includes(name));
  if (!found) throw new Error(`ไม่พบร้าน ${name} — รัน seed --reset ก่อนหรือยัง`);
  return found.id;
}

/** login ผ่านหน้า Keycloak จริง (บัญชีทดสอบใน realm: <user> / <user>-pass) */
export async function login(page: Page, user: string) {
  await page.goto("/me/bookings");
  await page.waitForURL(/keycloak\.jongyoung\.localhost/);
  await page.locator("#username").fill(user);
  await page.locator("#password").fill(`${user}-pass`);
  await page.locator("#kc-login").click();
  await page.waitForURL(/\/me\/bookings/);
}
