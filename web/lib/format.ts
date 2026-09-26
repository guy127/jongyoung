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

export type ChipState = "ok" | "low" | "full" | "closed";

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
