// เพิ่มการจองลงปฏิทิน 2 ทาง: ไฟล์ .ics (Apple/Outlook) และลิงก์ Google Calendar — ไม่ต้องใช้ API key

export type CalendarEvent = { uid: string; title: string; location: string; description: string; start: string; end: string };

/** "2026-10-10T23:30:00+07:00" → "20261010T163000Z" (รูปแบบเวลา UTC ของ iCalendar) */
const toUtcStamp = (iso: string) => new Date(iso).toISOString().replace(/[-:]/g, "").replace(/\.\d{3}/, "");

/** ข้อความใน .ics ต้อง escape \ ; , และขึ้นบรรทัดใหม่ (RFC 5545) */
const escape = (s: string) => s.replace(/\\/g, "\\\\").replace(/;/g, "\\;").replace(/,/g, "\\,").replace(/\n/g, "\\n");

export function icsContent(e: CalendarEvent): string {
  return [
    "BEGIN:VCALENDAR",
    "VERSION:2.0",
    "PRODID:-//jongyoung//booking//TH",
    "BEGIN:VEVENT",
    `UID:${e.uid}@jongyoung`,
    `DTSTAMP:${toUtcStamp(new Date().toISOString())}`,
    `DTSTART:${toUtcStamp(e.start)}`,
    `DTEND:${toUtcStamp(e.end)}`,
    `SUMMARY:${escape(e.title)}`,
    `LOCATION:${escape(e.location)}`,
    `DESCRIPTION:${escape(e.description)}`,
    "END:VEVENT",
    "END:VCALENDAR",
  ].join("\r\n");
}

export function googleCalendarUrl(e: CalendarEvent): string {
  const params = new URLSearchParams({
    action: "TEMPLATE",
    text: e.title,
    dates: `${toUtcStamp(e.start)}/${toUtcStamp(e.end)}`,
    location: e.location,
    details: e.description,
  });
  return `https://calendar.google.com/calendar/render?${params}`;
}

/** ให้ browser ดาวน์โหลดไฟล์ .ics */
export function downloadIcs(e: CalendarEvent) {
  const blob = new Blob([icsContent(e)], { type: "text/calendar;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `jongyoung-${e.uid.slice(0, 8)}.ics`;
  a.click();
  URL.revokeObjectURL(url);
}
