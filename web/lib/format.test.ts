import { describe, expect, it } from "vitest";
import { chipState, closedDaysLabel, dateKey, isClosedDay, defaultSearch, fmtTime, isGoogleMapsUrl, mapHref, nextDayLabel, ratingLabel, shiftDate } from "./format";

describe("chipState (ข้อ 9 เคส 27) — เทียบกับจำนวนคนที่เลือก ไม่ใช่ 1", () => {
  const seats = 10;
  it("ร้านปิด → closed", () => {
    expect(chipState({ available: 10, closed: true }, seats, 2)).toBe("closed");
  });
  it("ว่างน้อยกว่าจำนวนคน → full", () => {
    expect(chipState({ available: 3 }, seats, 4)).toBe("full");
  });
  it("ว่างพอแต่ ≤ 30% ของที่นั่ง → low", () => {
    expect(chipState({ available: 3 }, seats, 2)).toBe("low");
  });
  it("ว่างพอและ > 30% → ok", () => {
    expect(chipState({ available: 4 }, seats, 2)).toBe("ok");
  });
  it("available พอดีกับจำนวนคน → ยังจองได้", () => {
    expect(chipState({ available: 2 }, seats, 2)).toBe("low");
  });
});

describe("ratingLabel (ข้อ 9 เคส 28)", () => {
  it("ไม่มีรีวิว", () => {
    expect(ratingLabel({ average: null, count: 0 })).toEqual({ kind: "none", text: "ยังไม่มีรีวิว" });
  });
  it("1–4 รีวิว → รีวิวน้อย ไม่โชว์ดาวเด่น", () => {
    expect(ratingLabel({ average: 5, count: 1 })).toEqual({ kind: "few", text: "รีวิวน้อย (1)" });
  });
  it("≥ 5 รีวิว → ดาว + ทศนิยม 1 ตำแหน่ง", () => {
    expect(ratingLabel({ average: 4.8, count: 46 })).toEqual({ kind: "stars", text: "4.8 · 46 รีวิว" });
    expect(ratingLabel({ average: 5, count: 5 }).text).toBe("5.0 · 5 รีวิว");
  });
});

describe("nextDayLabel (ข้อ 9 เคส 29)", () => {
  it("ช่วงหลังเที่ยงคืนของรอบวันที่ 10 → (เช้าวันที่ 11)", () => {
    expect(nextDayLabel("2026-10-11T00:30:00+07:00", "2026-10-10")).toBe("(เช้าวันที่ 11)");
  });
  it("ช่วงในวันเดียวกัน → ไม่มีป้าย", () => {
    expect(nextDayLabel("2026-10-10T23:30:00+07:00", "2026-10-10")).toBe("");
  });
  it("ใช้เวลาไทยเสมอ แม้ timestamp มาเป็น UTC", () => {
    // 2026-10-10T17:30Z = 00:30 ของวันที่ 11 เวลาไทย
    expect(nextDayLabel("2026-10-10T17:30:00Z", "2026-10-10")).toBe("(เช้าวันที่ 11)");
  });
});

describe("defaultSearch", () => {
  it("ตอนนี้ 17:10 → 18:00 วันนี้ (ต้องล่วงหน้า 30 นาทีแล้วปัดขึ้น)", () => {
    expect(defaultSearch(new Date("2026-10-10T17:10:00+07:00"))).toEqual({ date: "2026-10-10", time: "18:00" });
  });
  it("ตอนนี้ 17:00 พอดี → 17:30", () => {
    expect(defaultSearch(new Date("2026-10-10T17:00:00+07:00"))).toEqual({ date: "2026-10-10", time: "17:30" });
  });
  it("ดึกเกิน → พรุ่งนี้เที่ยง", () => {
    expect(defaultSearch(new Date("2026-10-10T23:20:00+07:00"))).toEqual({ date: "2026-10-11", time: "12:00" });
  });
});

describe("เวลาและวันที่แบบไทย", () => {
  it("fmtTime แสดงเวลาไทย 24 ชม.", () => {
    expect(fmtTime("2026-10-10T11:00:00Z")).toBe("18:00");
  });
  it("dateKey ใช้วันที่ตามเวลาไทย", () => {
    expect(dateKey(new Date("2026-10-10T18:00:00Z"))).toBe("2026-10-11");
  });
  it("shiftDate เลื่อนวันแบบข้ามเดือน", () => {
    expect(shiftDate("2026-10-31", 1)).toBe("2026-11-01");
    expect(shiftDate("2026-10-01", -1)).toBe("2026-09-30");
  });
});

describe("วันปิดประจำสัปดาห์", () => {
  it("ป้ายเรียงจันทร์ก่อน และต่อคำแบบไทย", () => {
    expect(closedDaysLabel([])).toBe("");
    expect(closedDaysLabel([1])).toBe("ปิดทุกวันจันทร์");
    expect(closedDaysLabel([2, 1])).toBe("ปิดทุกวันจันทร์และอังคาร");
    expect(closedDaysLabel([0, 1, 2])).toBe("ปิดทุกวันจันทร์, อังคาร และอาทิตย์");
  });
  it("ดูจากวันทำการ (2026-10-12 = จันทร์, 0 = อาทิตย์ ตรงกับ Go)", () => {
    expect(isClosedDay([1], "2026-10-12")).toBe(true);
    expect(isClosedDay([1], "2026-10-13")).toBe(false);
    expect(isClosedDay([0], "2026-10-11")).toBe(true);
  });
});

describe("ลิงก์แผนที่", () => {
  it("มีลิงก์ที่เจ้าของร้านวาง → ใช้ลิงก์นั้น", () => {
    expect(mapHref("สยาม", "https://maps.app.goo.gl/AbC")).toBe("https://maps.app.goo.gl/AbC");
  });
  it("ไม่มี → ค้นจากที่อยู่", () => {
    expect(mapHref("สยาม พารากอน", "")).toBe("https://www.google.com/maps/search/?api=1&query=%E0%B8%AA%E0%B8%A2%E0%B8%B2%E0%B8%A1%20%E0%B8%9E%E0%B8%B2%E0%B8%A3%E0%B8%B2%E0%B8%81%E0%B8%AD%E0%B8%99");
  });
  it("รับเฉพาะลิงก์ Google Maps แบบ https", () => {
    expect(isGoogleMapsUrl("https://maps.app.goo.gl/AbC")).toBe(true);
    expect(isGoogleMapsUrl("https://www.google.com/maps/place/x")).toBe(true);
    expect(isGoogleMapsUrl("https://goo.gl/maps/AbC")).toBe(true);
    expect(isGoogleMapsUrl("javascript:alert(1)")).toBe(false);
    expect(isGoogleMapsUrl("http://maps.app.goo.gl/AbC")).toBe(false);
    expect(isGoogleMapsUrl("https://www.google.com/search?q=x")).toBe(false);
    expect(isGoogleMapsUrl("https://maps.app.goo.gl.evil.example/x")).toBe(false);
  });
});
