// รวมการแสดงเวลา/วันที่แบบไทย และกติกาการแสดงผลที่ต้องตรงกับ backend (CLAUDE.md ข้อ 8.4)
// เวลาทุกตัวจาก API เป็น timestamp เต็ม — แปลงเป็นเวลาไทยที่นี่ที่เดียว

export const TZ = "Asia/Bangkok";

const timeFmt = new Intl.DateTimeFormat("th-TH", { timeZone: TZ, hour: "2-digit", minute: "2-digit", hour12: false });
const shortDateFmt = new Intl.DateTimeFormat("th-TH", { timeZone: TZ, weekday: "short", day: "numeric", month: "short" });
const longDateFmt = new Intl.DateTimeFormat("th-TH", { timeZone: TZ, weekday: "long", day: "numeric", month: "short", year: "numeric" });
// en-CA ให้รูปแบบ YYYY-MM-DD พอดี ใช้ทำ key ของวัน (วันที่ตามเวลาไทย)
const keyFmt = new Intl.DateTimeFormat("en-CA", { timeZone: TZ, year: "numeric", month: "2-digit", day: "2-digit" });
const dayFmt = new Intl.DateTimeFormat("en-US", { timeZone: TZ, day: "numeric" });

type DateInput = string | Date;
const toDate = (d: DateInput) => (typeof d === "string" ? new Date(d) : d);

/** "18:00" */
export const fmtTime = (d: DateInput) => timeFmt.format(toDate(d));
/** "ส. 10 ต.ค." */
export const fmtShortDate = (d: DateInput) => shortDateFmt.format(toDate(d));
/** "วันเสาร์ที่ 10 ต.ค. 2569" */
export const fmtLongDate = (d: DateInput) => longDateFmt.format(toDate(d));

/** วันที่ตามเวลาไทยในรูป YYYY-MM-DD */
export const dateKey = (d: DateInput) => keyFmt.format(toDate(d));

/** "YYYY-MM-DD" → Date ตอนเที่ยงวันเวลาไทย (เที่ยงวันกันพลาดวันเวลาแปลง timezone) */
export const keyToDate = (key: string) => new Date(`${key}T12:00:00+07:00`);

/** เลื่อนวันที่ "YYYY-MM-DD" ไป n วัน */
export const shiftDate = (key: string, days: number) => dateKey(new Date(keyToDate(key).getTime() + days * 86_400_000));

export const todayKey = () => dateKey(new Date());

/**
 * ป้ายกำกับเวลาหลังเที่ยงคืนของรอบข้ามคืน เช่น "(เช้าวันที่ 11)"
 * businessDate = วันทำการของรอบ ถ้าวันตามปฏิทินของ slot ไม่ใช่วันเดียวกัน แปลว่าเป็นช่วงหลังเที่ยงคืน
 */
export function nextDayLabel(slotStart: DateInput, businessDate: string): string {
  if (dateKey(slotStart) === businessDate) return "";
  return `(เช้าวันที่ ${dayFmt.format(toDate(slotStart))})`;
}

/** "18:00–19:00" หรือ "23:30–00:30 (เช้าวันที่ 11)" */
export function fmtRange(start: DateInput, end: DateInput, businessDate: string): string {
  const label = nextDayLabel(end, businessDate);
  return `${fmtTime(start)}–${fmtTime(end)}${label ? " " + label : ""}`;
}

/** ชื่อวันตามเลข time.Weekday ของ Go (0 = อาทิตย์ … 6 = เสาร์) — ค่าเดียวกับ closed_weekdays จาก API */
export const WEEKDAYS = ["อาทิตย์", "จันทร์", "อังคาร", "พุธ", "พฤหัสบดี", "ศุกร์", "เสาร์"];
/** ลำดับแสดงผลเริ่มวันจันทร์ (ตามปฏิทินที่คนไทยคุ้น) */
export const WEEKDAY_ORDER = [1, 2, 3, 4, 5, 6, 0];

/** วันในสัปดาห์ของวันทำการ "YYYY-MM-DD" (keyToDate = เที่ยงวันเวลาไทย จึงเป็นวันเดียวกันใน UTC) */
export const weekdayOf = (businessDate: string) => keyToDate(businessDate).getUTCDay();

/** วันทำการนี้เป็นวันปิดประจำสัปดาห์ไหม — backend ตัดสินจริง ใช้ที่หน้าเว็บเพื่อบอกเหตุผลเท่านั้น */
export const isClosedDay = (closedWeekdays: number[], businessDate: string) => closedWeekdays.includes(weekdayOf(businessDate));

/** [1] → "ปิดทุกวันจันทร์", [1, 2] → "ปิดทุกวันจันทร์และอังคาร", [0, 1, 2] → "ปิดทุกวันจันทร์, อังคาร และอาทิตย์", [] → "" */
export function closedDaysLabel(closedWeekdays: number[]): string {
  const names = WEEKDAY_ORDER.filter((d) => closedWeekdays.includes(d)).map((d) => WEEKDAYS[d]);
  if (names.length === 0) return "";
  if (names.length === 1) return `ปิดทุกวัน${names[0]}`;
  return `ปิดทุกวัน${names.slice(0, -1).join(", ")}${names.length > 2 ? " " : ""}และ${names.at(-1)}`;
}

/** "11:00–22:00", "11:00–22:00 · พัก 15:00–17:00", "เปิด 24 ชม." — ป้ายเวลาเปิดของร้านทุกที่ใช้ตัวนี้ */
export function hoursLabel(r: { open_time: string; close_time: string; open_24h: boolean; break_start: string; break_end: string }): string {
  const base = r.open_24h ? "เปิด 24 ชม." : `${r.open_time}–${r.close_time}`;
  return r.break_start ? `${base} · พัก ${r.break_start}–${r.break_end}` : base;
}

/**
 * slot ถัดไปต่อจาก slot ก่อนหน้าสนิทไหม — ไม่ต่อ = มีช่วงพักร้านคั่น
 * (API ตัดช่วงพักออกจากลิสต์; ช่วงที่เลย lead time หายจากต้นลิสต์เท่านั้น จึงไม่ทำให้เกิดช่องว่างกลางลิสต์)
 */
export const isGap = (prev: { end_at: string } | undefined, next: { start_at: string }) =>
  !!prev && new Date(prev.end_at).getTime() !== new Date(next.start_at).getTime();

/** slot ที่ต่อกันสนิทนับจาก index start ไม่เกิน n ช่วง — หยุดเมื่อเจอช่วงพักหรือหมดลิสต์ */
export function contiguousFrom<T extends { start_at: string; end_at: string }>(slots: T[], start: number, n: number): T[] {
  const out: T[] = [];
  for (let i = start; i < slots.length && out.length < n; i++) {
    if (out.length > 0 && isGap(slots[i - 1], slots[i])) break;
    out.push(slots[i]);
  }
  return out;
}

export type ChipState ="ok" | "low" | "full" | "closed";

/**
 * สถานะปุ่มเวลา (ข้อ 8.4): เทียบที่ว่างกับจำนวนคนที่ผู้ใช้เลือก
 * เต็ม = ว่างไม่พอสำหรับกลุ่มนี้, เหลือน้อย = พอแต่ว่าง ≤ 30% ของที่นั่งทั้งร้าน
 */
export function chipState(slot: { available: number; closed?: boolean }, seats: number, party: number): ChipState {
  if (slot.closed) return "closed";
  if (slot.available < party) return "full";
  if (slot.available <= seats * 0.3) return "low";
  return "ok";
}

export type RatingLabel = { kind: "none" | "few" | "stars"; text: string };

/** รีวิว < 5 ไม่โชว์ดาวเด่น (สอดคล้องกับ Bayesian ที่ backend ใช้เรียงร้าน) */
export function ratingLabel(rating: { average: number | null; count: number }): RatingLabel {
  if (rating.count === 0 || rating.average === null) return { kind: "none", text: "ยังไม่มีรีวิว" };
  if (rating.count < 5) return { kind: "few", text: `รีวิวน้อย (${rating.count})` };
  return { kind: "stars", text: `${rating.average.toFixed(1)} · ${rating.count.toLocaleString("th-TH")} รีวิว` };
}

/**
 * ค่าเริ่มต้นของแถบค้นหา: วันนี้ + ช่วงครึ่งชั่วโมงแรกที่ยังจองทัน (ตอนนี้ + 30 นาที ปัดขึ้น)
 * ถ้าดึกเกิน 23:30 แล้ว → พรุ่งนี้เที่ยง
 */
export function defaultSearch(now: Date = new Date()): { date: string; time: string } {
  const [h, m] = fmtTime(now).split(":").map(Number);
  const earliest = h * 60 + m + 30;
  const rounded = Math.ceil(earliest / 30) * 30;
  if (rounded > 23 * 60 + 30) return { date: shiftDate(dateKey(now), 1), time: "12:00" };
  return { date: dateKey(now), time: minutesToClock(rounded) };
}

/** "18:00" → นาทีจากเที่ยงคืน */
export const clockToMinutes = (clock: string) => {
  const [h, m] = clock.split(":").map(Number);
  return h * 60 + m;
};

/** นาทีจากเที่ยงคืน → "18:00" (เกิน 24 ชม. วนกลับ) */
export const minutesToClock = (minutes: number) => {
  const m = ((minutes % 1440) + 1440) % 1440;
  return `${String(Math.floor(m / 60)).padStart(2, "0")}:${String(m % 60).padStart(2, "0")}`;
};

/** ลิงก์แผนที่ของร้าน: ใช้ลิงก์ Google Maps ที่เจ้าของร้านวางไว้ ถ้าไม่มีค่อยค้นจากที่อยู่ */
export const mapHref = (address: string, mapUrl?: string) =>
  mapUrl || `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(address)}`;

/** กติกาเดียวกับ isGoogleMapsURL ฝั่ง Go — ตรวจที่ฟอร์มเพื่อ UX เท่านั้น API ตรวจซ้ำเสมอ */
export function isGoogleMapsUrl(s: string): boolean {
  let u: URL;
  try {
    u = new URL(s);
  } catch {
    return false;
  }
  if (u.protocol !== "https:") return false;
  const isMapsPath = u.pathname === "/maps" || u.pathname.startsWith("/maps/");
  switch (u.hostname) {
    case "maps.app.goo.gl":
    case "maps.google.com":
    case "maps.google.co.th":
      return true;
    case "google.com":
    case "www.google.com":
    case "google.co.th":
    case "www.google.co.th":
      return isMapsPath;
    case "goo.gl":
      return u.pathname.startsWith("/maps/");
  }
  return false;
}
