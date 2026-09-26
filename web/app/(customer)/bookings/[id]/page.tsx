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
  const row = "grid grid-cols-[140px_1fr] gap-3 border-b border-border py-3.5 last:border-0";

  return (
    <section aria-labelledby="ok-h" className="mx-auto flex max-w-[760px] flex-col overflow-hidden rounded-3xl border border-border bg-surface">
      <div className="flex flex-col items-center gap-3 border-b border-border p-8 text-center">
        <span className={`grid size-16 place-items-center rounded-full ${cancelled ? "bg-full-weak text-full" : "bg-ok-weak text-ok"}`}>
          <CheckCircle2 size={32} aria-hidden />
        </span>
        <h1 id="ok-h" className="font-display text-[32px]">{cancelled ? "การจองนี้ถูกยกเลิกแล้ว" : "จองสำเร็จแล้ว"}</h1>
        {!cancelled && <p className="text-muted">แสดงเลขที่จองนี้ที่ร้านเมื่อไปถึง ดูย้อนหลังได้ที่หน้าการจองของฉัน</p>}
        <div className="flex flex-col items-center rounded-2xl bg-chip px-6 py-2">
          <span className="text-[13px] text-muted">เลขที่จอง</span>
          <span className="text-[28px] font-semibold tracking-wider tabular">{b.code}</span>
        </div>
      </div>

      <dl className="px-8 py-2">
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
              <span><strong>ได้ถึง {fmtShortDate(b.cancel_until)} {fmtTime(b.cancel_until)} น.</strong> หลังจากนั้นยกเลิกในระบบไม่ได้ หากไปไม่ได้โปรดแจ้งร้านโดยตรง</span>
            </dd>
          </div>
        )}
      </dl>

      <div className="flex flex-wrap gap-2 px-8 pb-8 pt-2">
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
