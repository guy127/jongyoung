"use client";

import { Clock, Minus, Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import { signIn, useSession } from "next-auth/react";
import { Fragment, useMemo, useState } from "react";

import { ChoiceButton } from "@/components/bases/Choice";
import { Field } from "@/components/bases/layout";
import TimeChip, { chipNote } from "@/components/bases/TimeChip";
import { Alert, Button, Skeleton } from "@/components/bases/ui";
import BookingError from "@/components/booking/BookingError";
import { apiError } from "@/lib/api";
import { chipState, closedDaysLabel, contiguousFrom, fmtRange, fmtShortDate, fmtTime, isGap, keyToDate, nextDayLabel, shiftDate, todayKey } from "@/lib/format";
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
  // เอาเฉพาะ slot ที่ต่อกันสนิท — ถ้าคร่อมช่วงพัก ห้ามกระโดดข้ามไปนับหลังพัก (ไม่งั้นส่งเวลาจบผิดไปหลายชั่วโมง)
  const covered = startIndex >= 0 ? contiguousFrom(slots, startIndex, duration) : [];
  const cut = startIndex >= 0 && covered.length < duration;
  const hitsBreak = cut && startIndex + covered.length < slots.length; // ยังมี slot ต่อ แต่ต่อไม่สนิท = ชนช่วงพัก
  const beyondClose = cut && !hitsBreak;
  const short = covered.find((s) => s.available < party);
  const ready = covered.length === duration && !short;
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
    <aside aria-labelledby="book-h" className="glass flex flex-col gap-6 rounded-3xl border border-border p-5 sm:p-6 lg:sticky lg:top-24">
      <h2 id="book-h" className="text-[22px] font-semibold">{editing ? `แก้ไขการจอง ${editing.code}` : "จองโต๊ะ"}</h2>

      <Field label="วันที่" id="date-l">
        {/* เลื่อนแนวนอนได้ถึงขอบแผง */}
        <div role="group" aria-labelledby="date-l" className="-mx-5 flex gap-2 overflow-x-auto px-5 pb-1 sm:-mx-6 sm:px-6">
          {days.map((d) => (
            <ChoiceButton key={d} active={d === date} onClick={() => pickDate(d)} className="rounded-xl">
              {fmtShortDate(keyToDate(d))}
            </ChoiceButton>
          ))}
        </div>
      </Field>

      <Field label="จำนวนคน" id="party-l" hint={`รับได้สูงสุด ${r.seats} คนต่อการจอง`}>
        <PartyStepper labelledBy="party-l" value={party} max={r.seats} onChange={setParty} />
      </Field>

      <Field label="เวลาเริ่ม" id="start-l">
        {availability.isLoading && <div className="grid grid-cols-4 gap-2">{Array.from({ length: 8 }, (_, i) => <Skeleton key={i} className="h-14" />)}</div>}
        {availability.isError && (
          <Alert tone="full" title="โหลดเวลาว่างไม่สำเร็จ">
            <button type="button" className="link" onClick={() => availability.refetch()}>ลองอีกครั้ง</button>
          </Alert>
        )}
        {availability.data && slots.length === 0 && (
          <p className="rounded-xl bg-chip p-4 text-sm">
            {availability.data.closed ? `วันนี้ร้านปิด (${closedDaysLabel(r.closed_weekdays)}) ลองเลือกวันอื่น` : "วันนี้ไม่มีช่วงที่ยังจองทันแล้ว ลองเลือกวันอื่น"}
          </p>
        )}
        {slots.length > 0 && (
          <div role="group" aria-labelledby="start-l" className="grid max-h-80 grid-cols-4 gap-2 overflow-y-auto p-0.5">
            {slots.map((s, i) => {
              const state = chipState(s, r.seats, party);
              return (
                <Fragment key={s.start_at}>
                  {isGap(slots[i - 1], s) && (
                    <p className="col-span-4 border-t border-dashed border-border-strong pt-2 text-[13px] text-muted tabular">
                      พักร้าน {fmtTime(slots[i - 1].end_at)}–{fmtTime(s.start_at)}
                    </p>
                  )}
                  <TimeChip time={fmtTime(s.start_at)} state={state} note={chipNote(state, s.available)}
                    dayLabel={nextDayLabel(s.start_at, date)} selected={i === startIndex}
                    inRange={startIndex >= 0 && i > startIndex && i < startIndex + covered.length}
                    onClick={() => { setStart(fmtTime(s.start_at)); setError(null); }} />
                </Fragment>
              );
            })}
          </div>
        )}
      </Field>

      <Field label="นานเท่าไหร่" id="dur-l">
        <div role="group" aria-labelledby="dur-l" className="grid grid-cols-4 gap-2">
          {durations.map((d) => (
            <ChoiceButton key={d.slots} tone="sel" active={d.slots === duration} onClick={() => setDuration(d.slots)} className="rounded-xl px-2">
              {d.label}
            </ChoiceButton>
          ))}
        </div>
      </Field>

      {start && startIndex < 0 && availability.data && <Alert tone="warn" title={`${start} จองไม่ได้แล้ว`}>เลือกเวลาอื่นจากด้านบน</Alert>}
      {beyondClose && <Alert tone="warn" title="เลยเวลาปิดร้าน">ลดระยะเวลา หรือเลือกเวลาเริ่มให้เร็วขึ้น</Alert>}
      {hitsBreak && <Alert tone="warn" title="ชนช่วงพักร้าน">ลดระยะเวลา หรือเลือกเวลาเริ่มหลังช่วงพัก</Alert>}
      {short && !cut && (
        <Alert tone="warn" title={`ช่วง ${fmtTime(short.start_at)} เหลือ ${short.available} ที่`}>ไม่พอสำหรับ {party} คน — ลดจำนวนคนหรือเลือกเวลาอื่น</Alert>
      )}

      {/* กติกายกเลิกต้องเห็นก่อนกดยืนยันเสมอ (ข้อ 8.4) */}
      {first && last && cancelUntil && (
        <BookingSummary date={date} start={first.start_at} end={last.end_at} party={party} cancelUntil={cancelUntil} cancelBefore={r.cancel_before_minutes} />
      )}

      {error && <BookingError error={error} />}

      <div className="flex flex-col gap-3">
        <Button variant="cta" onClick={submit} loading={save.isPending} disabled={status === "authenticated" && !ready} className="min-h-13 text-[17px]">
          {status !== "authenticated" ? "เข้าสู่ระบบเพื่อจอง" : save.isPending ? "กำลังจอง…" : editing ? "บันทึกการแก้ไข" : "ยืนยันการจอง"}
        </Button>
        <p className="text-[13px] leading-relaxed text-muted">ระบบตรวจที่นั่งอีกครั้งตอนกดยืนยัน ถ้ามีคนจองช่วงเดียวกันก่อน จะแจ้งที่นั่งที่เหลือจริงทันที</p>
      </div>
    </aside>
  );
}

/** ปุ่ม − จำนวน + (แตะได้ 44px ขึ้นไป) */
function PartyStepper({ labelledBy, value, max, onChange }: { labelledBy: string; value: number; max: number; onChange: (n: number) => void }) {
  const step = "chip-btn grid size-11 place-items-center rounded-lg border-transparent bg-chip";
  return (
    <div role="group" aria-labelledby={labelledBy} className="flex h-14 items-center justify-between rounded-xl border border-border-strong bg-surface px-1.5">
      <button type="button" aria-label="ลดจำนวนคน" disabled={value <= 1} onClick={() => onChange(value - 1)} className={step}><Minus size={18} /></button>
      <span aria-live="polite" className="text-[17px] font-semibold tabular">{value} คน</span>
      <button type="button" aria-label="เพิ่มจำนวนคน" disabled={value >= max} onClick={() => onChange(value + 1)} className={step}><Plus size={18} /></button>
    </div>
  );
}

/** สรุปสิ่งที่กำลังจะจอง + กติกายกเลิก */
function BookingSummary({ date, start, end, party, cancelUntil, cancelBefore }: {
  date: string; start: string; end: string; party: number; cancelUntil: Date; cancelBefore: number;
}) {
  return (
    <div className="flex flex-col gap-2 rounded-2xl border border-border bg-surface p-4">
      <p className="font-semibold tabular">{fmtShortDate(keyToDate(date))} · {fmtRange(start, end, date)} · {party} คน</p>
      <p className="flex gap-2 text-sm leading-relaxed text-muted">
        <Clock size={16} className="mt-1 shrink-0" aria-hidden />
        <span>
          ยกเลิกหรือแก้ไขได้ถึง <strong className="font-semibold text-text">{fmtShortDate(cancelUntil)} {fmtTime(cancelUntil)} น.</strong>{" "}
          (ก่อนเวลาจอง {cancelBefore} นาที)
        </span>
      </p>
    </div>
  );
}

function durationOf(b: Booking) {
  const minutes = (new Date(b.end_at).getTime() - new Date(b.start_at).getTime()) / 60_000;
  return Math.min(Math.max(Math.round(minutes / 30), 1), 8);
}
