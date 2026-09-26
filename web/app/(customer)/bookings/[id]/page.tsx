"use client";

import { CalendarPlus, CheckCircle2, Clock } from "lucide-react";
import { use } from "react";

import { Alert, LinkButton, Skeleton, buttonClass } from "@/components/bases/ui";
import { apiError } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { fmtLongDate, fmtRange, fmtShortDate, fmtTime, keyToDate } from "@/lib/format";
import { downloadIcs, googleCalendarUrl, type CalendarEvent } from "@/lib/ics";
import { useBooking } from "@/services/bookings";

// หน้ายืนยันการจอง — จุดที่ผู้ใช้ตัดสินว่าระบบน่าเชื่อถือ (CLAUDE.md ข้อ 8.4)
export default function BookingPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const { data: b, isLoading, error } = useBooking(id);

  if (isLoading) return <Skeleton className="mx-auto h-[520px] max-w-[760px]" />;
  if (error || !b) return <div className="mx-auto max-w-[760px]"><Alert tone="full" title={errorMessage(apiError(error))} /></div>;

  const event: CalendarEvent = {
    uid: b.id,
    title: `จองโต๊ะ ${b.restaurant.name} (${b.party_size} คน)`,
    location: b.restaurant.address,
    description: `เลขที่จอง ${b.code}`,
    start: b.start_at,
    end: b.end_at,
  };
  const cancelled = b.status === "cancelled";
  // มือถือ: ป้ายอยู่บนค่า / desktop: สองคอลัมน์
  const row = "grid gap-1 border-b border-border py-4 last:border-0 sm:grid-cols-[140px_1fr] sm:gap-4";

  return (
    <section aria-labelledby="ok-h" className="mx-auto flex max-w-[760px] flex-col overflow-hidden rounded-3xl border border-border bg-surface">
      <div className="flex flex-col items-center gap-4 border-b border-border px-5 py-8 text-center sm:p-10">
        <span className={`grid size-16 place-items-center rounded-full ${cancelled ? "bg-full-weak text-full" : "bg-ok-weak text-ok"}`}>
          <CheckCircle2 size={32} aria-hidden />
        </span>
        <h1 id="ok-h" className="font-display text-[28px] text-balance sm:text-[32px]">{cancelled ? "การจองนี้ถูกยกเลิกแล้ว" : "จองสำเร็จแล้ว"}</h1>
        {!cancelled && <p className="text-muted">แสดงเลขที่จองนี้ที่ร้านเมื่อไปถึง ดูย้อนหลังได้ที่หน้าการจองของฉัน</p>}
        <div className="mt-2 flex flex-col items-center gap-1 rounded-2xl bg-chip px-8 py-3">
          <span className="text-[13px] text-muted">เลขที่จอง</span>
          <span className="text-[28px] font-semibold tracking-wider tabular">{b.code}</span>
        </div>
      </div>

      <dl className="px-5 py-2 sm:px-10">
        <div className={row}><dt className="text-muted">ร้าน</dt><dd className="font-semibold">{b.restaurant.name}</dd></div>
        <div className={row}>
          <dt className="text-muted">วันและเวลา</dt>
          <dd className="font-semibold tabular">{fmtLongDate(keyToDate(b.business_date))} · {fmtRange(b.start_at, b.end_at, b.business_date)}</dd>
        </div>
        <div className={row}><dt className="text-muted">จำนวนคน</dt><dd className="font-semibold">{b.party_size} คน</dd></div>
        <div className={row}>
          <dt className="text-muted">ที่อยู่</dt>
          <dd className="flex flex-col gap-1">
            <span>{b.restaurant.address}</span>
            <a className="link" target="_blank" rel="noreferrer"
              href={`https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(b.restaurant.address)}`}>เปิดใน Google Maps</a>
          </dd>
        </div>
        {!cancelled && (
          <div className={row}>
            <dt className="text-muted">ยกเลิก/แก้ไข</dt>
            <dd className="flex gap-2">
              <Clock size={18} className="mt-1 shrink-0" aria-hidden />
              <span>
                <strong className="font-semibold">ได้ถึง {fmtShortDate(b.cancel_until)} {fmtTime(b.cancel_until)} น.</strong>
                <span className="text-muted"> หลังจากนั้นยกเลิกในระบบไม่ได้ หากไปไม่ได้โปรดแจ้งร้านโดยตรง</span>
              </span>
            </dd>
          </div>
        )}
      </dl>

      {/* มือถือ: ปุ่มเต็มความกว้างเรียงลง / desktop: เรียงในแถว */}
      <div className="flex flex-col gap-3 px-5 pb-6 pt-2 sm:flex-row sm:flex-wrap sm:px-10 sm:pb-10">
        {!cancelled && (
          <>
            <button type="button" className={buttonClass("cta")} onClick={() => downloadIcs(event)}>
              <CalendarPlus size={18} aria-hidden /> เพิ่มลงปฏิทิน (.ics)
            </button>
            <a className={buttonClass("outline")} href={googleCalendarUrl(event)} target="_blank" rel="noreferrer">Google Calendar</a>
          </>
        )}
        <LinkButton href="/me/bookings">ดูการจองของฉัน</LinkButton>
        {b.can_change && <LinkButton href={`/restaurants/${b.restaurant.id}?edit=${b.id}`}>แก้ไขการจอง</LinkButton>}
      </div>
    </section>
  );
}
