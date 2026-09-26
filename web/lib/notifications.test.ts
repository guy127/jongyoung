import { describe, expect, it } from "vitest";

import { bellLabel, notificationHref, notificationText } from "./notifications";
import type { Notification } from "./types";

const base: Notification = {
  id: "n1",
  kind: "booking_created",
  booking_id: "b1",
  restaurant_id: "r1",
  restaurant_name: "ท่าเรือซีฟู้ดบาร์",
  customer_name: "มะลิ วงศ์ดี",
  business_date: "2026-10-10",
  start_at: "2026-10-10T23:30:00+07:00",
  end_at: "2026-10-11T00:30:00+07:00",
  party_size: 3,
  reason: "",
  read_at: null,
  created_at: "2026-10-01T10:00:00+07:00",
};

describe("notificationText — ข้อความตาม kind", () => {
  it("ร้านยกเลิก → ชื่อร้าน + เวลา (มีป้ายข้ามวัน) + เหตุผล", () => {
    const text = notificationText({ ...base, kind: "booking_cancelled_by_restaurant", reason: "ไฟดับ" });
    expect(text).toContain("ท่าเรือซีฟู้ดบาร์ ยกเลิกการจองของคุณ");
    expect(text).toContain("23:30–00:30 (เช้าวันที่ 11)");
    expect(text).toContain("· ไฟดับ");
  });
  it("จองใหม่ → ชื่อลูกค้า + จำนวนคน", () => {
    expect(notificationText(base)).toContain("มะลิ วงศ์ดี จอง 3 คน");
  });
  it("แก้การจอง", () => {
    expect(notificationText({ ...base, kind: "booking_updated" })).toContain("มะลิ วงศ์ดี แก้การจองเป็น 3 คน");
  });
  it("ลูกค้ายกเลิก", () => {
    expect(notificationText({ ...base, kind: "booking_cancelled" })).toContain("มะลิ วงศ์ดี ยกเลิกการจอง");
  });
});

describe("notificationHref", () => {
  it("ลูกค้า → หน้าการจอง", () => {
    expect(notificationHref({ ...base, kind: "booking_cancelled_by_restaurant" })).toBe("/bookings/b1");
  });
  it("เจ้าของร้าน → บอร์ดของวันทำการ (ไม่ใช่วันปฏิทินของเวลาเริ่ม)", () => {
    expect(notificationHref(base)).toBe("/owner/bookings?restaurant=r1&date=2026-10-10");
  });
});

describe("bellLabel", () => {
  it("บอกจำนวนที่ยังไม่อ่านให้ screen reader", () => {
    expect(bellLabel(3)).toBe("การแจ้งเตือน 3 รายการที่ยังไม่อ่าน");
    expect(bellLabel(0)).toBe("การแจ้งเตือน");
  });
});
