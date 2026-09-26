import { defineConfig } from "@playwright/test";

// E2E ทดสอบระบบจริงทั้งก้อน: browser → Caddy → Next.js → Go → PostgreSQL + Keycloak
// รันด้วย `make e2e` (ล้าง seed ให้ข้อมูลเริ่มต้นเหมือนเดิมทุกครั้ง แล้วรันใน container ของ Playwright)
//
// PROXY_HOST: ใน container, Chromium ตีความ *.localhost เป็น 127.0.0.1 เองเสมอ
// จึงต้องบอกให้ส่งไปที่ caddy แทน (บนเครื่องเราไม่ต้องตั้ง เพราะ caddy อยู่ที่ 127.0.0.1 จริง)
const proxy = process.env.PROXY_HOST;

export default defineConfig({
  testDir: "./tests",
  outputDir: process.env.E2E_OUTPUT ?? "test-results",
  // ทุกเทสต์ใช้ฐานข้อมูลเดียวกัน → รันทีละตัวให้ผลคงที่ (ความถูกต้องตอนจองพร้อมกันทดสอบแล้วใน Go integration test)
  workers: 1,
  fullyParallel: false,
  retries: 0,
  reporter: [["list"]],
  timeout: 45_000,
  expect: { timeout: 10_000 },
  use: {
    baseURL: "http://jongyoung.localhost",
    locale: "th-TH",
    timezoneId: "Asia/Bangkok",
    trace: "retain-on-failure",
    launchOptions: proxy ? { args: [`--host-resolver-rules=MAP *.localhost ${proxy}`] } : undefined,
  },
  projects: [
    { name: "setup", testMatch: /auth\.setup\.ts/ },
    {
      name: "desktop",
      use: { viewport: { width: 1280, height: 900 } },
      dependencies: ["setup"],
      testIgnore: /auth\.setup\.ts/,
    },
    {
      // มือถือ 375px (ข้อ 8.6: mobile-first) — เฉพาะหน้าที่ไม่ต้อง login
      name: "mobile",
      use: { viewport: { width: 375, height: 812 }, isMobile: true, hasTouch: true },
      testMatch: /browse\.spec\.ts/,
    },
  ],
});
