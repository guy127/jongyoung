"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";

import { apiError } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { businessWindow, closureClockPoints, closureRangeLabel, fmtShortDate, fmtTime, isClosedDay, todayKey } from "@/lib/format";
import type { AffectedBooking, ApiError } from "@/lib/types";
import { type ClosureInput, useClosures, useCreateClosure, useDeleteClosure } from "@/services/closures";
import { useMe } from "@/services/me";
import { useRestaurant } from "@/services/restaurants";

const input = "h-9 rounded-md border border-border-strong bg-surface px-2.5 text-text";

export default function ClosuresPage() {
  return (
    <Suspense>
      <Closures />
    </Suspense>
  );
}

/**
 * ปิดร้านชั่วคราว (CLAUDE.md 5.1): รายการช่วงปิด + ฟอร์มสองแบบ
 * บันทึกแล้วได้ 409 CLOSURE_AFFECTS_BOOKINGS → แสดงการจองที่จะถูกยกเลิก แล้วยืนยันด้วยรายการ id ที่เห็นเท่านั้น
 */
function Closures() {
  const router = useRouter();
  const params = useSearchParams();
  const me = useMe();
  const restaurantId = params.get("restaurant") ?? me.data?.restaurants[0]?.id;
  const restaurant = useRestaurant(restaurantId);
  const closures = useClosures(restaurantId);
  const create = useCreateClosure(restaurantId);
  const remove = useDeleteClosure(restaurantId);

  const [mode, setMode] = useState<"partial" | "days">("partial");
  const [date, setDate] = useState(params.get("date") ?? todayKey());
  const [startOffset, setStartOffset] = useState(0);
  const [endOffset, setEndOffset] = useState(30);
  const [from, setFrom] = useState(todayKey());
  const [to, setTo] = useState(todayKey());
  const [reason, setReason] = useState("");
  const [affected, setAffected] = useState<AffectedBooking[] | null>(null);
  const [changed, setChanged] = useState(false); // ได้ 409 ตอนยืนยัน = มีการจองเข้ามาใหม่ระหว่างดูรายการ
  const [error, setError] = useState<ApiError | null>(null);
  const [done, setDone] = useState("");

  const hours = restaurant.data;
  const closedToday = hours ? isClosedDay(hours.closed_weekdays, date) : false;
  const pastDate = date < todayKey();
  const dateBlocked = pastDate || closedToday;

  // จุดเวลาในรอบเปิด-ปิดของวันทำการที่เลือกเท่านั้น (ข้อ 5.4) — กันเลือกเวลานอกรอบหรือผิดคืนของร้านข้ามเที่ยงคืน
  const points = hours ? closureClockPoints(hours, date) : [];
  const startOptions = points.slice(0, -1);
  const effectiveStart = startOptions.some((p) => p.offset === startOffset) ? startOffset : startOptions[0]?.offset ?? 0;
  const endOptions = points.filter((p) => p.offset > effectiveStart);
  const effectiveEnd = endOptions.some((p) => p.offset === endOffset) ? endOffset : endOptions[0]?.offset ?? effectiveStart + 30;
  const startPoint = points.find((p) => p.offset === effectiveStart);
  const endPoint = points.find((p) => p.offset === effectiveEnd);

  // ช่วงเวลาจริงที่จะปิด — แสดงเสมอก่อนกดบันทึก กันปิดผิดคืนแบบเงียบ ๆ (ร้านข้ามเที่ยงคืน)
  const preview: { start_at: string; end_at: string } | null =
    mode === "partial"
      ? startPoint && endPoint
        ? { start_at: startPoint.iso, end_at: endPoint.iso }
        : null
      : hours
        ? { start_at: businessWindow(hours, from).opensAt, end_at: businessWindow(hours, to).closesAt }
        : null;

  // แก้ฟอร์มแล้วรายการที่เห็นอาจไม่ตรงช่วงใหม่ → ต้องบันทึกใหม่ให้ API ตรวจอีกรอบ
  const edit = (set: (v: string) => void) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
    set(e.target.value);
    setAffected(null);
  };
  const editOffset = (set: (n: number) => void) => (e: React.ChangeEvent<HTMLSelectElement>) => {
    set(Number(e.target.value));
    setAffected(null);
  };

  const body = (): ClosureInput =>
    mode === "partial"
      ? { date, start_time: startPoint?.clock, end_time: endPoint?.clock, reason }
      : { from_date: from, to_date: to, reason };

  const submit = async (confirm?: AffectedBooking[]) => {
    setError(null);
    setDone("");
    try {
      await create.mutateAsync({ ...body(), confirm_booking_ids: confirm?.map((b) => b.id) });
      setDone(confirm?.length ? `ปิดร้านแล้ว — ยกเลิก ${confirm.length} การจองและแจ้งลูกค้าแล้ว` : "ปิดร้านแล้ว");
      setAffected(null);
      setChanged(false);
      setReason("");
    } catch (err) {
      const e = apiError(err);
      if (e?.code === "CLOSURE_AFFECTS_BOOKINGS") {
        setChanged(!!confirm);
        setAffected((e.details?.bookings as AffectedBooking[]) ?? []);
        return;
      }
      setError(e ?? { code: "NETWORK", message: "" });
    }
  };

  const reopen = (id: string) => {
    if (window.confirm("เปิดร้านกลับช่วงนี้? การจองที่ถูกยกเลิกไปแล้วจะไม่กลับมา")) remove.mutate(id);
  };

  if (me.data && me.data.restaurants.length === 0) {
    return <p className="rounded-md border border-dashed border-border-strong bg-surface p-6 text-center">ยังไม่มีร้าน — เพิ่มร้านที่หน้า “ร้านของฉัน” ก่อน</p>;
  }
  const guests = affected?.reduce((sum, b) => sum + b.party_size, 0) ?? 0;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-lg font-semibold">ปิดร้านชั่วคราว</h1>
        <select aria-label="ร้าน" value={restaurantId ?? ""} onChange={(e) => router.replace(`/owner/closures?restaurant=${e.target.value}`)} className={input}>
          {me.data?.restaurants.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
        </select>
      </div>

      <section aria-labelledby="list-h" className="border border-border bg-surface">
        <h2 id="list-h" className="border-b border-border px-3.5 py-2.5 font-semibold">ช่วงปิดที่ยังไม่จบ</h2>
        {closures.data?.length === 0 && <p className="p-4 text-muted">ไม่มีช่วงปิด — ร้านเปิดตามเวลาปกติ</p>}
        <ul>
          {closures.data?.map((c) => (
            <li key={c.id} className="flex flex-wrap items-center gap-3 border-t border-border px-3.5 py-2 first:border-0">
              <span className="font-medium tabular">{closureRangeLabel(c)}</span>
              <span className="text-muted">{c.reason}</span>
              <button type="button" onClick={() => reopen(c.id)} disabled={remove.isPending} className="obtn obtn-sm ml-auto">เปิดร้านกลับ</button>
            </li>
          ))}
        </ul>
      </section>

      <form onSubmit={(e) => { e.preventDefault(); void submit(); }} className="flex flex-col gap-4 border border-border bg-surface p-4">
        <fieldset className="flex gap-2">
          <legend className="sr-only">รูปแบบการปิด</legend>
          {([["partial", "บางช่วงของวัน"], ["days", "ทั้งวันหรือหลายวัน"]] as const).map(([key, label]) => (
            <label key={key} className={`inline-flex min-h-9 items-center gap-1.5 rounded-md border px-3 ${mode === key ? "border-text font-semibold" : "border-border-strong"}`}>
              <input type="radio" name="mode" checked={mode === key} onChange={() => { setMode(key); setAffected(null); }} />
              {label}
            </label>
          ))}
        </fieldset>

        {mode === "partial" ? (
          <div className="grid gap-3 sm:grid-cols-3">
            <label className="flex flex-col gap-1 font-medium">รอบของวันที่<input type="date" value={date} onChange={edit(setDate)} className={input} /></label>
            <label className="flex flex-col gap-1 font-medium">
              ตั้งแต่
              <select value={effectiveStart} onChange={editOffset(setStartOffset)} disabled={dateBlocked || !hours} className={input}>
                {startOptions.map((p) => <option key={p.offset} value={p.offset}>{p.label}</option>)}
              </select>
            </label>
            <label className="flex flex-col gap-1 font-medium">
              ถึง
              <select value={effectiveEnd} onChange={editOffset(setEndOffset)} disabled={dateBlocked || !hours} className={input}>
                {endOptions.map((p) => <option key={p.offset} value={p.offset}>{p.label}</option>)}
              </select>
            </label>
            <p className="text-muted sm:col-span-3">เลือกคืนที่ร้านเปิด เวลาหลังเที่ยงคืนจะนับรวมในคืนนั้น</p>
            {dateBlocked && (
              <p role="alert" className="rounded-md border border-full-line bg-full-weak p-3 text-full sm:col-span-3">
                {pastDate ? "วันที่เลือกผ่านไปแล้ว" : "วันนี้เป็นวันปิดประจำสัปดาห์ของร้านอยู่แล้ว ไม่ต้องปิดเพิ่ม"}
              </p>
            )}
          </div>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="flex flex-col gap-1 font-medium">จากวัน<input type="date" value={from} onChange={edit(setFrom)} className={input} /></label>
            <label className="flex flex-col gap-1 font-medium">ถึงวัน<input type="date" value={to} onChange={edit(setTo)} className={input} /></label>
          </div>
        )}

        {preview && (
          <p className="rounded-md border border-border-strong bg-chip px-3 py-2 font-medium tabular">
            จะปิด: {closureRangeLabel(preview)}
          </p>
        )}

        <label className="flex flex-col gap-1 font-medium">
          เหตุผล * <span className="font-normal text-muted">ลูกค้าจะเห็นข้อความนี้</span>
          <textarea required maxLength={200} rows={2} value={reason} onChange={edit(setReason)} className={`${input} h-auto py-2`} placeholder="เช่น ไฟดับทั้งซอย, ปิดปรับปรุงร้าน" />
        </label>

        {!affected && (
          <button
            type="submit"
            disabled={create.isPending || (mode === "partial" && (dateBlocked || !preview))}
            aria-busy={create.isPending || undefined}
            className="obtn obtn-primary self-start"
          >
            บันทึกช่วงปิด
          </button>
        )}

        {affected && (
          <section aria-labelledby="affected-h" className="flex flex-col gap-3 rounded-md border border-full-line bg-full-weak p-3.5">
            <h2 id="affected-h" className="font-semibold text-full">การจองที่จะถูกยกเลิก {affected.length} รายการ (รวม {guests} คน)</h2>
            {changed && <p role="status">มีการจองเข้ามาใหม่ระหว่างนี้ — รายการด้านล่างเป็นข้อมูลล่าสุด</p>}
            <table className="w-full border-collapse bg-surface tabular">
              <thead className="text-muted"><tr><th className="px-2 py-1.5 text-left font-medium">เลขที่</th><th className="px-2 text-left font-medium">ลูกค้า</th><th className="px-2 text-left font-medium">เวลา</th><th className="px-2 text-left font-medium">คน</th></tr></thead>
              <tbody>
                {affected.map((b) => (
                  <tr key={b.id} className="border-t border-border">
                    <td className="px-2 py-1.5">{b.code}</td><td className="px-2">{b.customer_name}</td>
                    <td className="px-2">{fmtShortDate(b.start_at)} {fmtTime(b.start_at)}–{fmtTime(b.end_at)}</td><td className="px-2">{b.party_size}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            <p className="text-muted">ลูกค้าจะได้รับแจ้งเตือนพร้อมเหตุผล — ควรโทรแจ้งด้วยถ้าเป็นการปิดกะทันหัน</p>
            <div className="flex gap-2">
              <button type="button" onClick={() => void submit(affected)} disabled={create.isPending} aria-busy={create.isPending || undefined} className="obtn obtn-danger">
                ยืนยันปิดร้านและยกเลิก {affected.length} การจอง
              </button>
              <button type="button" onClick={() => setAffected(null)} className="obtn">ไม่ปิดแล้ว</button>
            </div>
          </section>
        )}

        {error && <p role="alert" className="rounded-md border border-full-line bg-full-weak p-3 text-full">✕ {errorMessage(error.code === "NETWORK" ? null : error)} {error.code !== "NETWORK" && `(${error.code})`}</p>}
        {done && <p role="status" className="rounded-md border border-ok-line bg-ok-weak p-3 text-ok">✓ {done}</p>}
      </form>
    </div>
  );
}
