import { describe, expect, it } from "vitest";

import { googleCalendarUrl, icsContent } from "./ics";

const event = {
  uid: "abc",
  title: "จองโต๊ะ ครัวบ้านสวน (2 คน)",
  location: "ซอยสุขุมวิท 49, กรุงเทพฯ",
  description: "เลขที่จอง JY-7F3K2A",
  start: "2026-10-10T23:30:00+07:00",
  end: "2026-10-11T00:30:00+07:00",
};

describe("ics", () => {
  it("เวลาเป็น UTC (Z) ตามมาตรฐาน iCalendar และ escape เครื่องหมายจุลภาค", () => {
    const ics = icsContent(event);
    expect(ics).toContain("DTSTART:20261010T163000Z");
    expect(ics).toContain("DTEND:20261010T173000Z");
    expect(ics).toContain("LOCATION:ซอยสุขุมวิท 49\\, กรุงเทพฯ");
    expect(ics.split("\r\n")[0]).toBe("BEGIN:VCALENDAR");
  });

  it("ลิงก์ Google Calendar ใส่ช่วงเวลาเป็น UTC", () => {
    const url = new URL(googleCalendarUrl(event));
    expect(url.searchParams.get("dates")).toBe("20261010T163000Z/20261010T173000Z");
    expect(url.searchParams.get("text")).toBe(event.title);
  });
});
