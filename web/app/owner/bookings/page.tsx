"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Fragment, Suspense } from "react";

import { bookingStatusLabel, fmtLongDate, fmtRange, gapLabel, isGap, keyToDate, nextDayLabel, shiftDate, fmtTime, todayKey } from "@/lib/format";
import { useBoard } from "@/services/bookings";
import { useClosures } from "@/services/closures";
import { useMe } from "@/services/me";

export default function OwnerBookingsPage() {
  return (
    <Suspense>
      <Board />
    </Suspense>
  );
}

/** บอร์ดรายวันทำการ: seat bar ต่อช่วง 30 นาที + ตารางการจอง (เก็บร้าน/วันไว้ใน URL) */
function Board() {
  const router = useRouter();
  const params = useSearchParams();
  const me = useMe();
  const restaurantId = params.get("restaurant") ?? me.data?.restaurants[0]?.id;
  const date = params.get("date") ?? todayKey();
  const board = useBoard(restaurantId, date);
  const closures = useClosures(restaurantId);
  const go = (patch: Record<string, string>) =>
    router.replace(`/owner/bookings?${new URLSearchParams({ restaurant: restaurantId ?? "", date, ...patch })}`);

  const active = board.data?.bookings.filter((b) => b.status === "active") ?? [];
  const cancelled = (board.data?.bookings.length ?? 0) - active.length;
  const guests = active.reduce((sum, b) => sum + b.party_size, 0);
  const peak = board.data?.slots.reduce((best, s) => (s.booked > best.booked ? s : best), { booked: 0, start_at: "", end_at: "" });
  const seats = board.data?.seats ?? 0;

  // "ปิดร้านตอนนี้" โผล่เมื่อรอบที่กำลังดูยังไม่จบ และเป็นรอบวันนี้ หรือรอบที่กำลังเปิดอยู่แล้ว
  // (ร้านข้ามเที่ยงคืน เช่น 18:00–02:00 ดูบอร์ดของเมื่อวานตอนตี 1 ก็ยังนับว่า "กำลังเปิด")
  const nowMs = new Date().getTime();
  const opensMs = board.data ? new Date(board.data.opens_at).getTime() : 0;
  const closesMs = board.data ? new Date(board.data.closes_at).getTime() : 0;
  const canCloseNow = !!board.data && !board.data.closed && nowMs < closesMs && (date === todayKey() || nowMs >= opensMs);
  // เริ่มปิดจาก "ตอนนี้ปัดลง :00/:30" หรือเวลาเปิด แล้วแต่อันไหนช้ากว่า (กันเริ่มปิดก่อนร้านเปิด)
  const roundedNowMs = Math.floor(nowMs / (30 * 60 * 1000)) * 30 * 60 * 1000;
  const closeNowStart = fmtTime(new Date(Math.max(roundedNowMs, opensMs)));

  if (me.data && me.data.restaurants.length === 0) {
    return <p className="rounded-md border border-dashed border-border-strong bg-surface p-6 text-center">ยังไม่มีร้าน — เพิ่มร้านที่หน้า “ร้านของฉัน” ก่อน</p>;
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <select aria-label="ร้าน" value={restaurantId ?? ""} onChange={(e) => go({ restaurant: e.target.value })}
          className="h-9 rounded-md border border-border-strong bg-surface px-2.5">
          {me.data?.restaurants.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
        </select>
        <div role="group" aria-label="วันทำการ" className="flex items-center rounded-md border border-border-strong bg-surface">
          <button type="button" aria-label="วันก่อนหน้า" onClick={() => go({ date: shiftDate(date, -1) })} className="grid size-9 place-items-center rounded-l-md transition-colors hover:bg-chip"><ChevronLeft size={18} /></button>
          <label className="px-2 font-semibold">
            <span className="sr-only">เลือกวัน</span>
            <input type="date" value={date} onChange={(e) => e.target.value && go({ date: e.target.value })} className="bg-transparent" />
          </label>
          <button type="button" aria-label="วันถัดไป" onClick={() => go({ date: shiftDate(date, 1) })} className="grid size-9 place-items-center rounded-r-md transition-colors hover:bg-chip"><ChevronRight size={18} /></button>
        </div>
        {board.data && (
          <span className="text-muted">
            รอบ{fmtLongDate(keyToDate(date))} · {board.data.closed
              ? <strong className="font-semibold text-text">วันปิดประจำสัปดาห์ — ไม่รับจอง</strong>
              : <>{fmtTime(board.data.opens_at)}–{fmtTime(board.data.closes_at)} {nextDayLabel(board.data.closes_at, date)}</>} · {seats} ที่นั่ง
          </span>
        )}
        {canCloseNow && (
          // ปิดกะทันหัน: กรอกวันทำการที่กำลังดู + เวลาเริ่มปิดถึงเวลาปิดร้านไว้ให้
          <Link className="obtn obtn-sm ml-auto"
            href={`/owner/closures?${new URLSearchParams({ restaurant: restaurantId ?? "", date,
              start: closeNowStart, end: fmtTime(board.data!.closes_at) })}`}>
            ปิดร้านตอนนี้
          </Link>
        )}
      </div>

      {board.isLoading && <p className="text-muted">กำลังโหลด…</p>}
      {board.isError && <p role="alert" className="text-full">โหลดบอร์ดไม่สำเร็จ <button type="button" className="link" onClick={() => board.refetch()}>ลองอีกครั้ง</button></p>}

      {board.data && (
        <>
          <section aria-label="สรุป" className="grid grid-cols-2 border border-border bg-surface sm:grid-cols-4">
            {[
              ["การจองที่ยืนยัน", String(active.length)],
              ["ลูกค้ารวม", `${guests} คน`],
              ["peak ของรอบ", peak && peak.booked > 0 ? `${peak.booked}/${seats} · ${fmtTime(peak.start_at)}` : "—"],
              ["ยกเลิกแล้ว", String(cancelled)],
            ].map(([label, value]) => (
              <div key={label} className="border-border p-3 not-last:border-r">
                <div className="text-muted">{label}</div>
                <div className="text-[22px] font-semibold tabular">{value}</div>
              </div>
            ))}
          </section>

          <div className="grid gap-4 lg:grid-cols-[420px_1fr] lg:items-start">
            {/* seat bar อยู่ฝั่ง owner: เจ้าของร้านอยากเห็นว่าคืนนี้แน่นช่วงไหน */}
            <section aria-labelledby="occ-h" className="border border-border bg-surface">
              <h2 id="occ-h" className="flex justify-between border-b border-border px-3.5 py-2.5 font-semibold">
                คนในร้านต่อช่วง 30 นาที <span className="font-normal text-muted">จอง/ที่นั่ง</span>
              </h2>
              <ul className="px-3.5 py-2">
                {board.data.slots.map((s, i, all) => {
                  const ratio = seats ? s.booked / seats : 0;
                  const color = ratio >= 1 ? "bg-full" : ratio >= 0.7 ? "bg-warn" : "bg-ok";
                  return (
                    <Fragment key={s.start_at}>
                      {isGap(all[i - 1], s) && (
                        <li className="grid h-6 grid-cols-[92px_1fr] items-center gap-2.5 text-muted tabular">
                          <span>{fmtTime(all[i - 1].end_at)}–{fmtTime(s.start_at)}</span><span>{gapLabel(all[i - 1].end_at, s.start_at, closures.data ?? []).split(" ")[0]}</span>
                        </li>
                      )}
                      <li className="grid h-6 grid-cols-[92px_1fr_48px] items-center gap-2.5 tabular">
                        <span className="text-muted">{fmtTime(s.start_at)} <span className="text-[12px]">{nextDayLabel(s.start_at, date) ? "+1" : ""}</span></span>
                        <span className="flex h-3 bg-chip"><span className={color} style={{ width: `${Math.min(ratio, 1) * 100}%` }} /></span>
                        <span className={`text-right ${ratio >= 0.7 ? "font-semibold" : "text-muted"}`}>{s.booked}/{seats}</span>
                      </li>
                    </Fragment>
                  );
                })}
              </ul>
              <p className="flex gap-3 border-t border-border px-3.5 py-2 text-muted">
                <span className="text-ok">■ &lt; 70%</span><span className="text-warn">■ ≥ 70%</span><span className="text-full">■ เต็ม</span><span>+1 = เช้าวันถัดไป</span>
              </p>
            </section>

            <section aria-labelledby="list-h" className="overflow-x-auto border border-border bg-surface">
              <h2 id="list-h" className="border-b border-border px-3.5 py-2.5 font-semibold">รายการจอง (เรียงตามเวลาเริ่ม)</h2>
              {board.data.bookings.length === 0 ? (
                <p className="p-4 text-muted">ยังไม่มีการจองในรอบนี้</p>
              ) : (
                <table className="w-full border-collapse tabular">
                  <thead className="bg-chip/50 text-muted">
                    <tr><th className="px-3.5 py-2 text-left font-medium">เวลา</th><th className="px-2 text-left font-medium">ลูกค้า</th><th className="px-2 text-left font-medium">คน</th><th className="px-2 text-left font-medium">เลขที่จอง</th><th className="px-3.5 text-left font-medium">สถานะ</th></tr>
                  </thead>
                  <tbody>
                    {board.data.bookings.map((b) => (
                      <tr key={b.id} className={`border-t border-border transition-colors hover:bg-chip/50 ${b.status === "cancelled" ? "text-muted" : ""}`}>
                        <td className="px-3.5 py-2 font-medium">{fmtRange(b.start_at, b.end_at, date)}</td>
                        <td className="px-2">{b.customer_name}</td>
                        <td className="px-2">{b.party_size}</td>
                        <td className="px-2 text-muted">{b.code}</td>
                        <td className={`px-3.5 ${b.status === "cancelled" ? "text-full" : "text-ok"}`}>{b.status === "cancelled" ? bookingStatusLabel(b, false) : "✓ ยืนยัน"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </section>
          </div>
        </>
      )}
    </div>
  );
}
