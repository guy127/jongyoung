"use client";

import { Minus, Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import { signIn, useSession } from "next-auth/react";
import { useMemo, useState } from "react";

import TimeChip, { chipNote } from "@/components/bases/TimeChip";
import { Alert, Skeleton } from "@/components/bases/ui";
import BookingError from "@/components/booking/BookingError";
import { apiError } from "@/lib/api";
import { chipState, fmtRange, fmtShortDate, fmtTime, keyToDate, nextDayLabel, shiftDate, todayKey } from "@/lib/format";
import type { ApiError, Booking, Restaurant, Slot } from "@/lib/types";
import { useSaveBooking } from "@/services/bookings";
import { useAvailability } from "@/services/restaurants";

const durations = [
  { slots: 1, label: "30 น." },
  { slots: 2, label: "1 ชม." },
  { slots: 3, label: "1.5 ชม." },
  { slots: 4, label: "2 ชม." },
];

type Props = {
  restaurant: Restaurant;
  initialDate: string;
  initialTime?: string;
  initialParty: number;
  editing?: Booking; // แก้ไขการจองเดิม → PUT แทน POST
};

/** แผงจอง: เลือกวัน/จำนวนคน/เวลา/ระยะเวลา → เห็นกติกายกเลิกก่อนกด → ยืนยัน */
export default function BookingPanel({ restaurant: r, initialDate, initialTime, initialParty, editing }: Props) {
  const router = useRouter();
  const { status } = useSession();
  const [date, setDate] = useState(initialDate);
  const [party, setParty] = useState(initialParty);
  const [start, setStart] = useState<string | undefined>(initialTime);
  const [duration, setDuration] = useState(editing ? durationOf(editing) : 2);
  const [error, setError] = useState<ApiError | null>(null);
  const availability = useAvailability(r.id, date);
  const save = useSaveBooking();

  // ตอนแก้ไข: ที่นั่งของการจองเดิมคืนกลับมาให้ตัวเองใช้ได้ (server ก็ไม่นับตัวเองเหมือนกัน — excludeID)
  const slots: Slot[] = useMemo(() => {
    const list = availability.data?.slots ?? [];
    if (!editing) return list;
    return list.map((s) =>
      s.start_at >= editing.start_at && s.start_at < editing.end_at ? { ...s, available: s.available + editing.party_size } : s,
    );
  }, [availability.data, editing]);

  const startIndex = slots.findIndex((s) => fmtTime(s.start_at) === start);
  const covered = startIndex >= 0 ? slots.slice(startIndex, startIndex + duration) : [];
  const beyondClose = startIndex >= 0 && startIndex + duration > slots.length;
  const short = covered.find((s) => s.available < party);
  const ready = covered.length === duration && !short && !beyondClose;
  const first = covered[0];
  const last = covered[covered.length - 1];
  const cancelUntil = first ? new Date(new Date(first.start_at).getTime() - r.cancel_before_minutes * 60_000) : null;

  const days = Array.from({ length: 7 }, (_, i) => shiftDate(todayKey(), i));

  const pickDate = (d: string) => {
    setDate(d);
    setStart(undefined);
    setError(null);
  };

  const submit = async () => {
    if (status !== "authenticated") return void signIn("keycloak");
    if (!ready || !first || !last || save.isPending) return;
    setError(null);
    try {
      const booking = await save.mutateAsync({
        id: editing?.id,
        input: { restaurant_id: r.id, date, start_time: fmtTime(first.start_at), end_time: fmtTime(last.end_at), party_size: party },
      });
      router.push(`/bookings/${booking.id}`);
    } catch (err) {
      const e = apiError(err);
      setError(e ?? { code: "NETWORK", message: "" });
      // ที่นั่งเปลี่ยนไปแล้ว → โหลดเวลาว่างล่าสุดให้เห็นของจริง
      if (e?.code === "NOT_ENOUGH_SEATS") void availability.refetch();
    }
  };

  return (
    <aside aria-labelledby="book-h" className="glass flex flex-col gap-4 rounded-3xl border border-border p-5 lg:sticky lg:top-24">
      <h2 id="book-h" className="text-[22px] font-semibold">{editing ? `แก้ไขการจอง ${editing.code}` : "จองโต๊ะ"}</h2>

      <div className="flex flex-col gap-1.5">
        <span className="text-sm font-medium">วันที่</span>
        <div role="group" aria-label="เลือกวัน" className="flex gap-1.5 overflow-x-auto pb-1">
          {days.map((d) => (
            <button key={d} type="button" onClick={() => pickDate(d)} aria-pressed={d === date}
              className={`min-h-12 shrink-0 rounded-xl border px-3 text-sm ${d === date ? "border-text bg-text text-surface" : "border-border bg-surface"}`}>
              {fmtShortDate(keyToDate(d))}
            </button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <span id="party-l" className="text-sm font-medium">จำนวนคน</span>
        <div role="group" aria-labelledby="party-l" className="flex h-12 items-center justify-between rounded-xl border border-border-strong bg-surface px-1">
          <button type="button" aria-label="ลดจำนวนคน" disabled={party <= 1} onClick={() => setParty(party - 1)} className="grid size-10 place-items-center rounded-lg bg-chip disabled:opacity-40"><Minus size={18} /></button>
          <span aria-live="polite" className="font-semibold tabular">{party} คน</span>
          <button type="button" aria-label="เพิ่มจำนวนคน" disabled={party >= r.seats} onClick={() => setParty(party + 1)} className="grid size-10 place-items-center rounded-lg bg-chip disabled:opacity-40"><Plus size={18} /></button>
        </div>
        <span className="text-[13px] text-soft">รับได้สูงสุด {r.seats} คนต่อการจอง</span>
      </div>

      <div className="flex flex-col gap-1.5">
        <span className="text-sm font-medium">เวลาเริ่ม</span>
        {availability.isLoading && <div className="grid grid-cols-4 gap-1.5">{Array.from({ length: 8 }, (_, i) => <Skeleton key={i} className="h-14" />)}</div>}
        {availability.isError && <Alert tone="full" title="โหลดเวลาว่างไม่สำเร็จ"><button type="button" className="underline" onClick={() => availability.refetch()}>ลองอีกครั้ง</button></Alert>}
        {availability.data && slots.length === 0 && <p className="rounded-xl bg-chip p-3 text-sm">วันนี้ไม่มีช่วงที่ยังจองทันแล้ว ลองเลือกวันอื่น</p>}
        {slots.length > 0 && (
          <div role="group" aria-label="เวลาเริ่ม" className="grid max-h-72 grid-cols-4 gap-1.5 overflow-y-auto pr-1">
            {slots.map((s, i) => {
              const state = chipState(s, r.seats, party);
              return (
                <TimeChip key={s.start_at} time={fmtTime(s.start_at)} state={state} note={chipNote(state, s.available)}
                  dayLabel={nextDayLabel(s.start_at, date)} selected={i === startIndex}
                  inRange={startIndex >= 0 && i > startIndex && i < startIndex + duration}
                  onClick={() => { setStart(fmtTime(s.start_at)); setError(null); }} />
              );
            })}
          </div>
        )}
      </div>

      <div className="flex flex-col gap-1.5">
        <span id="dur-l" className="text-sm font-medium">นานเท่าไหร่</span>
        <div role="group" aria-labelledby="dur-l" className="grid grid-cols-4 gap-1.5">
          {durations.map((d) => (
            <button key={d.slots} type="button" aria-pressed={d.slots === duration} onClick={() => setDuration(d.slots)}
              className={`min-h-11 rounded-xl border text-sm ${d.slots === duration ? "border-sel bg-sel text-sel-text" : "border-border-strong bg-surface"}`}>
              {d.label}
            </button>
          ))}
        </div>
      </div>

      {start && startIndex < 0 && availability.data && <Alert tone="warn" title={`${start} จองไม่ได้แล้ว`}>เลือกเวลาอื่นจากด้านบน</Alert>}
      {beyondClose && <Alert tone="warn" title="เลยเวลาปิดร้าน">ลดระยะเวลา หรือเลือกเวลาเริ่มให้เร็วขึ้น</Alert>}
      {short && !beyondClose && (
        <Alert tone="warn" title={`ช่วง ${fmtTime(short.start_at)} เหลือ ${short.available} ที่`}>ไม่พอสำหรับ {party} คน — ลดจำนวนคนหรือเลือกเวลาอื่น</Alert>
      )}

      {first && last && (
        <div className="flex flex-col gap-1 rounded-2xl border border-border bg-surface p-4">
          <span className="font-semibold tabular">{fmtShortDate(keyToDate(date))} · {fmtRange(first.start_at, last.end_at, date)} · {party} คน</span>
          {cancelUntil && (
            <span className="text-sm leading-relaxed">
              <strong>ยกเลิกหรือแก้ไขได้ถึง {fmtShortDate(cancelUntil)} {fmtTime(cancelUntil)} น.</strong> (ก่อนเวลาจอง {r.cancel_before_minutes} นาที) หลังจากนั้นยกเลิกในระบบไม่ได้
            </span>
          )}
        </div>
      )}

      {error && <BookingError error={error} />}

      <button type="button" onClick={submit} disabled={(status === "authenticated" && !ready) || save.isPending}
        className="cta min-h-13 rounded-full py-3.5 text-[17px] font-semibold disabled:cursor-not-allowed disabled:opacity-50">
        {status !== "authenticated" ? "เข้าสู่ระบบเพื่อจอง" : save.isPending ? "กำลังจอง…" : editing ? "บันทึกการแก้ไข" : "ยืนยันการจอง"}
      </button>
      <p className="text-[13px] leading-relaxed text-muted">ระบบตรวจที่นั่งอีกครั้งตอนกดยืนยัน ถ้ามีคนจองช่วงเดียวกันก่อน จะแจ้งที่นั่งที่เหลือจริงทันที</p>
    </aside>
  );
}

function durationOf(b: Booking) {
  const minutes = (new Date(b.end_at).getTime() - new Date(b.start_at).getTime()) / 60_000;
  return Math.min(Math.max(Math.round(minutes / 30), 1), 8);
}
