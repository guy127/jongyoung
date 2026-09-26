"use client";

import Link from "next/link";

import { fmtShortDate, fmtTime, keyToDate } from "@/lib/format";
import { useNextAvailable } from "@/services/restaurants";

/** empty state ของร้านที่เต็ม: ไม่ใช่จอเปล่า แต่เสนอวันถัดไปที่ว่าง (โหลดเฉพาะการ์ดที่เต็มเท่านั้น) */
export default function NextAvailableHint({ restaurantId, date, time, party }: { restaurantId: string; date: string; time: string; party: number }) {
  const { data, isLoading, isError } = useNextAvailable(restaurantId, date, time, party, true);
  if (isLoading) return <span className="text-[13px] text-soft">กำลังหาวันที่ว่าง…</span>;
  if (isError) return null;
  if (!data) return <span className="text-[13px] text-soft">ไม่มีช่วงว่างใน 14 วันข้างหน้าสำหรับ {party} คน</span>;
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <span className="text-[13px] text-muted">ว่างวันถัดไป:</span>
      {data.slots.slice(0, 3).map((s) => (
        <Link key={s.start_at} href={`/restaurants/${restaurantId}?date=${data.business_date}&time=${fmtTime(s.start_at)}&party_size=${party}`}
          className="rounded-lg border border-border-strong bg-surface px-2.5 py-1.5 text-sm tabular">
          {fmtShortDate(keyToDate(data.business_date))} {fmtTime(s.start_at)}
        </Link>
      ))}
    </div>
  );
}
