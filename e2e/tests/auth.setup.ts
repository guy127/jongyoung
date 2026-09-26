import { test as setup } from "@playwright/test";

import { AUTH, login } from "./helpers";

// login ครั้งเดียวแล้วเก็บ cookie ไว้ให้เทสต์อื่นใช้ (ไม่ต้อง login ใหม่ทุกเทสต์)
setup("login ลูกค้า (customer1)", async ({ page }) => {
  await login(page, "customer1");
  await page.context().storageState({ path: AUTH.customer });
});

setup("login เจ้าของร้าน (owner2)", async ({ page }) => {
  await login(page, "owner2");
  await page.context().storageState({ path: AUTH.owner });
});
