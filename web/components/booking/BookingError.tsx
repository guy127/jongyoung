import Link from "next/link";

import { Alert } from "@/components/bases/ui";
import { errorMessage } from "@/lib/errors";
import type { ApiError } from "@/lib/types";

/**
 * ผลของการจองที่ไม่สำเร็จ — 409 สองแบบต้องแยกกันชัด เพราะผู้ใช้ต้องทำคนละอย่าง (CLAUDE.md ข้อ 5.2)
 * - DUPLICATE_BOOKING: สีกลาง + ลิงก์ไปรายการเดิม และห้ามบอกว่าร้านเต็ม
 * - RESTAURANT_CLOSED: สีกลาง — ร้านปิดชั่วคราว ไม่ใช่ความผิดพลาดและไม่ใช่ "เต็ม"
 * - NOT_ENOUGH_SEATS และอื่น ๆ: กล่องแดง (หน้าร้าน refetch เวลาว่างให้แล้ว)
 */
export default function BookingError({ error }: { error: ApiError }) {
  if (error.code === "DUPLICATE_BOOKING") {
    return (
      <Alert tone="info" title="คุณมีการจองที่ทับช่วงนี้อยู่แล้ว">
        <Link href={`/bookings/${String(error.details?.booking_id)}`} className="link font-medium">ดูรายการจองเดิม</Link>
      </Alert>
    );
  }
  if (error.code === "RESTAURANT_CLOSED") {
    return (
      <Alert tone="info" title={errorMessage(error)}>
        เวลาว่างด้านบนอัปเดตแล้ว ลองเลือกช่วงอื่นหรือวันอื่น
      </Alert>
    );
  }
  return (
    <Alert tone="full" title={errorMessage(error.code === "NETWORK" ? null : error)}>
      {error.code === "NOT_ENOUGH_SEATS" && "เวลาว่างด้านบนอัปเดตเป็นข้อมูลล่าสุดแล้ว ลองเลือกช่วงอื่น"}
    </Alert>
  );
}
