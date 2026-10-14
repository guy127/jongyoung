import { describe, expect, it } from "vitest";
import { bookingStatusLabel, businessWindow, chipState, closedDaysLabel, closureClockPoints, closureEndOptions, closureRangeLabel, closureStartOptions, contiguousFrom, gapLabel, hoursLabel, isGap, dateKey, isClosedDay, defaultSearch, fmtTime, isGoogleMapsUrl, mapHref, nextDayLabel, ratingLabel, shiftDate } from "./format";

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

describe("hoursLabel", () => {
  const base = { open_time: "11:00", close_time: "22:00", open_24h: false, break_start: "", break_end: "" };
  it("ไม่มีช่วงพัก", () => {
    expect(hoursLabel(base)).toBe("11:00–22:00");
  });
  it("มีช่วงพัก", () => {
    expect(hoursLabel({ ...base, break_start: "15:00", break_end: "17:00" })).toBe("11:00–22:00 · พัก 15:00–17:00");
  });
  it("24 ชม.", () => {
    expect(hoursLabel({ ...base, open_time: "00:00", close_time: "00:00", open_24h: true })).toBe("เปิด 24 ชม.");
  });
});

describe("ช่วงพักในลิสต์ slot", () => {
  const slot = (start: string, end: string) => ({ start_at: `2026-10-10T${start}:00+07:00`, end_at: `2026-10-10T${end}:00+07:00` });
  // 13:00 13:30 | พัก | 17:00 17:30
  const slots = [slot("13:00", "13:30"), slot("13:30", "14:00"), slot("17:00", "17:30"), slot("17:30", "18:00")];

  it("isGap: ต่อกันสนิท = ไม่ใช่ช่วงพัก, ไม่ต่อ = ช่วงพัก, ตัวแรก = ไม่ใช่", () => {
    expect(isGap(slots[0], slots[1])).toBe(false);
    expect(isGap(slots[1], slots[2])).toBe(true);
    expect(isGap(undefined, slots[0])).toBe(false);
  });
  it("contiguousFrom: เลือก 13:00 นาน 1 ชม. → 2 ช่วงครบ", () => {
    expect(contiguousFrom(slots, 0, 2)).toEqual([slots[0], slots[1]]);
  });
  it("contiguousFrom: เลือก 13:30 นาน 2 ชม. → หยุดที่ช่วงพัก ได้ช่วงเดียว (ห้ามกระโดดไป 17:00)", () => {
    expect(contiguousFrom(slots, 1, 4)).toEqual([slots[1]]);
  });
  it("contiguousFrom: เลยท้ายลิสต์ → ได้เท่าที่มี", () => {
    expect(contiguousFrom(slots, 3, 2)).toEqual([slots[3]]);
  });
});

describe("bookingStatusLabel — ร้านยกเลิกต้องบอกว่าใครยกเลิกและเพราะอะไร", () => {
  const b = { status: "cancelled", cancelled_by: null, cancel_reason: "" } as const;
  it("ร้านยกเลิก → เหตุผล", () => {
    expect(bookingStatusLabel({ ...b, cancelled_by: "restaurant", cancel_reason: "ไฟดับ" }, false)).toBe("✕ ร้านยกเลิก · ไฟดับ");
  });
  it("ลูกค้ายกเลิกเอง / ข้อมูลเก่า", () => {
    expect(bookingStatusLabel({ ...b, cancelled_by: "customer" }, false)).toBe("✕ ยกเลิกแล้ว");
    expect(bookingStatusLabel(b, false)).toBe("✕ ยกเลิกแล้ว");
  });
  it("ยังไม่ยกเลิก", () => {
    expect(bookingStatusLabel({ ...b, status: "active" }, false)).toBe("✓ ยืนยันแล้ว");
    expect(bookingStatusLabel({ ...b, status: "active" }, true)).toBe("ไปแล้ว");
  });
});

describe("gapLabel — ช่องว่างระหว่าง slot เป็นช่วงพักหรือร้านปิด", () => {
  const closures = [{ id: "c1", start_at: "2026-10-10T18:00:00+07:00", end_at: "2026-10-10T20:00:00+07:00", reason: "ไฟดับ" }];
  it("ทับช่วงปิด → ร้านปิด + เหตุผล", () => {
    expect(gapLabel("2026-10-10T18:00:00+07:00", "2026-10-10T20:00:00+07:00", closures)).toBe("ร้านปิด 18:00–20:00 · ไฟดับ");
  });
  it("ไม่ทับ → พักร้าน", () => {
    expect(gapLabel("2026-10-10T14:00:00+07:00", "2026-10-10T17:00:00+07:00", closures)).toBe("พักร้าน 14:00–17:00");
  });
});

describe("closureClockPoints / closureStartOptions / closureEndOptions — dropdown ฟอร์มปิดร้านชั่วคราว (owner/closures) ต้องไม่ให้เลือกเวลานอกรอบ", () => {
  const overnight = { open_time: "18:00", close_time: "02:00", break_start: "", break_end: "", open_24h: false };
  const withBreak = { open_time: "11:00", close_time: "22:00", break_start: "15:00", break_end: "17:00", open_24h: false };
  const h24 = { open_time: "00:00", close_time: "00:00", break_start: "", break_end: "", open_24h: true };

  it("ร้านข้ามเที่ยงคืน 18:00–02:00: ไม่มี 17:00/03:00 — จุดสุดท้ายคือ 02:00 ของเช้าวันถัดไป", () => {
    const points = closureClockPoints(overnight, "2026-10-10");
    const clocks = points.map((p) => p.clock);
    expect(clocks).not.toContain("17:00");
    expect(clocks).not.toContain("03:00");
    expect(clocks[0]).toBe("18:00");
    expect(clocks.at(-1)).toBe("02:00");
    expect(points.at(-1)!.label).toBe("02:00 (เช้าวันที่ 11)");
    expect(points[0].label).toBe("18:00"); // ไม่ข้ามคืน ไม่มีป้าย
  });

  it("ร้านมีช่วงพัก 15:00–17:00: ข้ามเวลาในช่วงพัก แต่ยังเลือกขอบพักได้ (15:00 และ 17:00)", () => {
    const clocks = closureClockPoints(withBreak, "2026-10-10").map((p) => p.clock);
    expect(clocks).toContain("15:00");
    expect(clocks).toContain("17:00");
    expect(clocks).not.toContain("15:30");
    expect(clocks).not.toContain("16:00");
    expect(clocks).not.toContain("16:30");
  });

  it("ร้าน 24 ชม.: วนกลับมาจบที่ 00:00 ของเช้าวันถัดไป (ไม่ใช่ของวันเดียวกัน)", () => {
    const points = closureClockPoints(h24, "2026-10-10");
    expect(points[0]).toMatchObject({ clock: "00:00", label: "00:00" });
    expect(points.at(-1)).toMatchObject({ clock: "00:00", label: "00:00 (เช้าวันที่ 11)" });
    expect(points.at(-1)!.iso).toBe("2026-10-11T00:00:00+07:00");
  });

  it("closureStartOptions ตัดจุดสุดท้าย (เวลาปิด) ออก — เริ่มที่นั่นไม่มีช่วงให้ปิด", () => {
    const starts = closureStartOptions(overnight, "2026-10-10");
    expect(starts.at(-1)!.clock).not.toBe("02:00");
    expect(starts.at(-1)!.clock).toBe("01:30");
  });

  it("closureEndOptions คืนเฉพาะจุดหลัง start — ร้าน 24 ชม. เลือก start=23:30 แล้ว end มีแค่ 00:00 (เช้าวันถัดไป)", () => {
    const starts = closureStartOptions(h24, "2026-10-10");
    const last = starts.at(-1)!; // 23:30
    expect(last.clock).toBe("23:30");
    const ends = closureEndOptions(h24, "2026-10-10", last.offset);
    expect(ends).toHaveLength(1);
    expect(ends[0]).toMatchObject({ clock: "00:00", label: "00:00 (เช้าวันที่ 11)" });
  });

  it("closureEndOptions ของร้านปกติ 11:00–22:00 เลือก start=14:00 → ไม่มีเวลาก่อนหรือเท่ากับ 14:00", () => {
    const normal = { open_time: "11:00", close_time: "22:00", break_start: "", break_end: "", open_24h: false };
    const starts = closureStartOptions(normal, "2026-10-10");
    const start1400 = starts.find((p) => p.clock === "14:00")!;
    const ends = closureEndOptions(normal, "2026-10-10", start1400.offset);
    expect(ends.every((p) => p.offset > start1400.offset)).toBe(true);
    expect(ends[0].clock).toBe("14:30");
    expect(ends.at(-1)!.clock).toBe("22:00");
  });
});

describe("businessWindow — ใช้ทำกล่องสรุปวันเวลาจริงก่อนยืนยันปิดร้าน (โหมดทั้งวันหรือหลายวัน)", () => {
  it("ร้านข้ามเที่ยงคืน 18:00–02:00: closesAt ของวันทำการ 10 คือ 02:00 ของวันที่ 11", () => {
    const overnight = { open_time: "18:00", close_time: "02:00", break_start: "", break_end: "", open_24h: false };
    expect(businessWindow(overnight, "2026-10-10")).toEqual({ opensAt: "2026-10-10T18:00:00+07:00", closesAt: "2026-10-11T02:00:00+07:00" });
  });

  it("ปิดหลายวัน 10–12 ของร้านข้ามเที่ยงคืน → กล่องสรุปบอกวันจริงถูกต้องด้วย closureRangeLabel", () => {
    const overnight = { open_time: "18:00", close_time: "02:00", break_start: "", break_end: "", open_24h: false };
    const from = businessWindow(overnight, "2026-10-10");
    const to = businessWindow(overnight, "2026-10-12");
    const label = closureRangeLabel({ start_at: from.opensAt, end_at: to.closesAt });
    expect(label).toContain("10 ต.ค. 18:00");
    expect(label).toContain("13 ต.ค. 02:00"); // รอบของวันที่ 12 ปิดเช้าวันที่ 13
  });
});

describe("closureRangeLabel", () => {
  it("วันเดียวกัน → วันที่ครั้งเดียว", () => {
    expect(closureRangeLabel({ start_at: "2026-10-10T18:00:00+07:00", end_at: "2026-10-10T20:00:00+07:00" })).toMatch(/10 ต\.ค\. 18:00–20:00$/);
  });
  it("คนละวัน → ใส่วันที่ทั้งสองฝั่ง", () => {
    const label = closureRangeLabel({ start_at: "2026-10-20T17:00:00+07:00", end_at: "2026-10-22T23:00:00+07:00" });
    expect(label).toContain("20 ต.ค. 17:00");
    expect(label).toContain("22 ต.ค. 23:00");
  });
});
