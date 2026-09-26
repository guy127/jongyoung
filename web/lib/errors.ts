import { fmtShortDate, fmtTime } from "./format";
import type { ApiError } from "./types";

// หน้าเว็บตัดสินใจจาก error.code เท่านั้น แล้วแปลเป็นข้อความไทยที่นี่ (ไม่พึ่ง message จาก API)
const messages: Record<string, string> = {
  INVALID_REQUEST: "ข้อมูลไม่ครบหรือไม่ถูกต้อง",
  INVALID_TIME_RANGE: "เวลาสิ้นสุดต้องหลังเวลาเริ่ม",
  NOT_ON_HALF_HOUR: "เลือกเวลาได้เฉพาะ :00 หรือ :30",
  INVALID_DURATION: "จองได้ครั้งละ 30 นาทีถึง 4 ชั่วโมง",
  BOOKING_IN_PAST: "เวลานี้ผ่านไปแล้ว",
  TOO_LATE_TO_BOOK: "ต้องจองล่วงหน้าอย่างน้อย 30 นาที",
  TOO_FAR_AHEAD: "จองล่วงหน้าได้ไม่เกิน 90 วัน",
  INVALID_PARTY_SIZE: "จำนวนคนต้องอย่างน้อย 1 คน",
  PARTY_TOO_LARGE: "จำนวนคนมากกว่าที่นั่งทั้งร้าน",
  OUTSIDE_OPENING_HOURS: "ช่วงที่เลือกอยู่นอกเวลาเปิด–ปิดของร้าน",
  NOT_ENOUGH_SEATS: "ที่นั่งไม่พอแล้ว — ยังไม่ได้จองให้",
  DUPLICATE_BOOKING: "คุณมีการจองที่ทับช่วงนี้อยู่แล้ว",
  BOOKING_CANCELLED: "การจองนี้ถูกยกเลิกไปแล้ว",
  BOOKING_ALREADY_STARTED: "การจองนี้เริ่มหรือผ่านไปแล้ว",
  CANCEL_WINDOW_PASSED: "เลยเวลาที่แก้ไขหรือยกเลิกได้แล้ว",
  NOT_BOOKING_OWNER: "ไม่ใช่การจองของคุณ",
  NOT_OWNER: "เฉพาะเจ้าของร้านเท่านั้น",
  SEATS_BELOW_EXISTING_BOOKINGS: "ลดที่นั่งไม่ได้ มีการจองที่ใช้ที่นั่งมากกว่านี้",
  HOURS_CONFLICT_EXISTING_BOOKINGS: "เปลี่ยนเวลาไม่ได้ มีการจองที่อยู่นอกเวลาใหม่",
  IMAGE_REQUIRED: "ร้านต้องมีรูปอย่างน้อย 1 รูป",
  INVALID_OPENING_HOURS: "เวลาเปิด–ปิดต้องลงที่ :00 หรือ :30",
  OWN_RESTAURANT: "เจ้าของร้านรีวิวร้านตัวเองไม่ได้",
  REVIEW_EXISTS: "คุณรีวิวร้านนี้แล้ว แก้ไขรีวิวเดิมแทนได้",
  INVALID_REVIEW: "คะแนนต้องเป็น 1–5",
  NOT_FOUND: "ไม่พบข้อมูล",
  UNAUTHORIZED: "เซสชันหมดอายุ กรุณาเข้าสู่ระบบใหม่",
  RATE_LIMITED: "ทำรายการถี่เกินไป รอสักครู่แล้วลองใหม่",
};

/** ข้อความไทยของ error พร้อมรายละเอียดที่ช่วยให้ผู้ใช้ตัดสินใจต่อได้ */
export function errorMessage(err: ApiError | null): string {
  if (!err) return "เชื่อมต่อไม่สำเร็จ ตรวจสอบอินเทอร์เน็ตแล้วลองอีกครั้ง";
  const base = messages[err.code] ?? "เกิดข้อผิดพลาด ลองอีกครั้ง";
  const d = err.details ?? {};
  switch (err.code) {
    case "NOT_ENOUGH_SEATS":
      return `${base} — ช่วง ${fmtTime(String(d.at))} เหลือ ${d.available} ที่`;
    case "CANCEL_WINDOW_PASSED":
      return `${base} (ได้ถึง ${fmtShortDate(String(d.cancel_until))} ${fmtTime(String(d.cancel_until))} น.)`;
    case "SEATS_BELOW_EXISTING_BOOKINGS":
      return `${base} — ${fmtShortDate(String(d.at))} ${fmtTime(String(d.at))} มีคนในร้าน ${d.peak} คน`;
    case "TOO_LATE_TO_BOOK":
      return `${base} (เร็วสุด ${fmtTime(String(d.earliest_start_at))} น.)`;
    default:
      return base;
  }
}
