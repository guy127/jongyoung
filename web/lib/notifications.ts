// ข้อความและลิงก์ของแจ้งเตือน — API ส่งแค่ kind + snapshot ของการจอง หน้าเว็บประกอบข้อความไทยที่นี่ที่เดียว
import { fmtRange, fmtShortDate, keyToDate } from "./format";
import type { Notification } from "./types";

export function notificationText(n: Notification): string {
  const when = `${fmtShortDate(keyToDate(n.business_date))} ${fmtRange(n.start_at, n.end_at, n.business_date)}`;
  switch (n.kind) {
    case "booking_cancelled_by_restaurant":
      return `${n.restaurant_name} ยกเลิกการจองของคุณ ${when}${n.reason ? ` · ${n.reason}` : ""}`;
    case "booking_created":
      return `${n.customer_name} จอง ${n.party_size} คน ${when}`;
    case "booking_updated":
      return `${n.customer_name} แก้การจองเป็น ${n.party_size} คน ${when}`;
    case "booking_cancelled":
      return `${n.customer_name} ยกเลิกการจอง ${when}`;
  }
}

/** กดแล้วไปไหน: ลูกค้าไปหน้าการจอง, เจ้าของร้านไปบอร์ดของวันทำการนั้น (ร้านข้ามคืน วันทำการ ≠ วันปฏิทิน) */
export function notificationHref(n: Notification): string {
  if (n.kind === "booking_cancelled_by_restaurant") return `/bookings/${n.booking_id}`;
  return `/owner/bookings?restaurant=${n.restaurant_id}&date=${n.business_date}`;
}

export const bellLabel = (unread: number) => (unread > 0 ? `การแจ้งเตือน ${unread} รายการที่ยังไม่อ่าน` : "การแจ้งเตือน");
