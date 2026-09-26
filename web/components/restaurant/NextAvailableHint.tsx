"use client";

import Link from "next/link";

import { fmtShortDate, fmtTime, keyToDate, nextDayLabel } from "@/lib/format";
import { useNextAvailable } from "@/services/restaurants";

/**
 * empty state ของร้านที่เต็ม/ปิดช่วงที่ค้น: ไม่ใช่จอเปล่า แต่เสนอช่วงที่ยังว่าง (โหลดเฉพาะการ์ดที่จองไม่ได้เท่านั้น)
 * API คืนวันเดียวกับที่ค้นได้ เมื่อร้านแค่ยังไม่เปิดตอนเวลาที่ค้น (เช่น ค้น 10:30 ที่ร้านเปิด 17:00)
 */
export default function NextAvailableHint({ restaurantId, date, time, party }: { restaurantId: string; date: string; time: string; party: number }) {
  const { data, isLoading, isError } = useNextAvailable(restaurantId, date, time, party, true);
  if (isLoading) return <span className="text-[13px] text-soft">กำลังหาวันที่ว่าง…</span>;
  if (isError) return null;
  if (!data) return <span className="text-[13px] text-soft">ไม่มีช่วงว่างใน 14 วันข้างหน้าสำหรับ {party} คน</span>;
  const sameDay = data.business_date === date;
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <span className="text-[13px] text-muted">{sameDay ? "ช่วงที่ยังว่าง:" : `ว่างวันถัดไป ${fmtShortDate(keyToDate(data.business_date))}:`}</span>
      {data.slots.slice(0, 3).map((s) => (
        <Link key={s.start_at} href={`/restaurants/${restaurantId}?date=${data.business_date}&time=${fmtTime(s.start_at)}&party_size=${party}`}
          className="time-chip rounded-lg border border-ok-line bg-ok-weak px-2.5 py-1.5 text-sm font-medium text-ok tabular">
          {fmtTime(s.start_at)} {nextDayLabel(s.start_at, data.business_date)}
        </Link>
      ))}
    </div>
  );
}
