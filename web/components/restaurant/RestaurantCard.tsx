import Image from "next/image";
import Link from "next/link";

import Rating from "@/components/bases/Rating";
import TimeChip, { chipNote } from "@/components/bases/TimeChip";
import { chipState, closedDaysLabel, fmtTime, isClosedDay, nextDayLabel } from "@/lib/format";
import type { ListItem } from "@/lib/types";

import NextAvailableHint from "./NextAvailableHint";

type Props = { restaurant: ListItem; date: string; time: string; party: number };

/** การ์ดร้าน (พื้นทึบ ไม่ใช้ glass) + ปุ่มเวลา 5 ช่วงรอบเวลาที่ค้น — กดแล้วไปหน้าร้านพร้อมเวลาที่เลือกไว้ */
export default function RestaurantCard({ restaurant: r, date, time, party }: Props) {
  const detail = `/restaurants/${r.id}`;
  const slots = r.slots ?? [];
  const chips = slots.map((s) => {
    const state = chipState(s, r.seats, party);
    const clock = fmtTime(s.start_at);
    return { s, state, clock, dayLabel: nextDayLabel(s.start_at, date) };
  });
  const anyBookable = chips.some((c) => c.state === "ok" || c.state === "low");
  const allClosed = chips.length > 0 && chips.every((c) => c.state === "closed");
  const cover = r.images[0]?.url;

  return (
    <article className="card flex flex-col overflow-hidden rounded-2xl">
      <Link href={`${detail}?date=${date}&time=${time}&party_size=${party}`} className="relative block aspect-[16/8] bg-chip">
        {cover && <Image src={cover} alt={`รูปร้าน ${r.name}`} fill sizes="(min-width: 1024px) 360px, (min-width: 640px) 50vw, 100vw" className="object-cover" />}
        {r.cuisine && <span className="absolute left-3 top-3 rounded-full bg-black/55 px-2.5 py-0.5 text-[13px] text-white">{r.cuisine}</span>}
      </Link>
      <div className="flex flex-1 flex-col gap-4 p-5">
        <div className="flex flex-col gap-1.5">
          <Link href={`${detail}?date=${date}&time=${time}&party_size=${party}`} className="card-title text-[19px] font-semibold">{r.name}</Link>
          <div className="flex flex-wrap items-center gap-1.5 text-sm text-muted tabular">
            <Rating rating={r.rating} />
            <span>· {r.open_24h ? "เปิด 24 ชม." : `${r.open_time}–${r.close_time}`}</span>
          </div>
          {r.overnight && <span className="text-[13px] text-muted">เปิดถึง {r.close_time} ของเช้าวันถัดไป</span>}
          {r.closed_weekdays.length > 0 && <span className="text-[13px] text-muted">{closedDaysLabel(r.closed_weekdays)}</span>}
        </div>

        {anyBookable && (
          // แถวเลื่อนแนวนอน: ปุ่มกว้างคงที่ ข้อความไม่ตัดบรรทัด (การ์ดแคบกว่า 5 ปุ่มแทบทุกขนาดจอ)
          // ขยายเต็มขอบการ์ด (-mx-5 px-5) ให้เลื่อนได้สุดขอบ ปุ่มที่โผล่ครึ่งเดียวบอกว่ายังมีต่อ
          <div role="group" aria-label={`เวลาว่างของ ${r.name}`} className="-mx-5 mt-auto flex snap-x gap-2 overflow-x-auto px-5 pb-1">
            {chips.map(({ s, state, clock, dayLabel }) => (
              <TimeChip key={s.start_at} time={clock} state={state} note={chipNote(state, s.available)} dayLabel={dayLabel}
                href={`${detail}?date=${date}&time=${clock}&party_size=${party}`} className="w-[4.5rem] shrink-0 snap-start" />
            ))}
          </div>
        )}
        {!anyBookable && slots.length > 0 && (
          <div className="mt-auto flex flex-col gap-3 rounded-xl border border-dashed border-border-strong p-4">
            <span className="text-sm font-medium">{isClosedDay(r.closed_weekdays, date) ? `ร้านปิดวันนี้ (${closedDaysLabel(r.closed_weekdays)})` : allClosed ? "ร้านปิดช่วงเวลานี้" : "เต็มช่วงเวลานี้"}</span>
            <NextAvailableHint restaurantId={r.id} date={date} time={time} party={party} />
          </div>
        )}
      </div>
    </article>
  );
}
