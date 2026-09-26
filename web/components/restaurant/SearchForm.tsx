import { Search } from "lucide-react";

import { todayKey } from "@/lib/format";

const times = Array.from({ length: 48 }, (_, i) => `${String(Math.floor(i / 2)).padStart(2, "0")}:${i % 2 ? "30" : "00"}`);
const inputCls = "h-12 w-full rounded-xl border border-border-strong bg-surface px-3 text-base text-text";
const labelCls = "flex flex-col gap-2 text-sm font-medium";

type Props = { q: string; date: string; time: string; party: number; sort: string; cuisine: string };

/** แถบค้นหา (งานหลักของหน้าแรก) — form แบบ GET ธรรมดา ส่งค่าผ่าน URL ไม่ต้องใช้ JavaScript */
export default function SearchForm({ q, date, time, party, sort, cuisine }: Props) {
  return (
    <section aria-labelledby="hero-h" className="glass flex flex-col gap-6 rounded-3xl border border-border p-6 sm:p-8">
      <div className="flex flex-col gap-2">
        {/* text-balance + ไม่ตัดคำ "หมด" → ไม่มีคำเดียวหล่นไปบรรทัดใหม่ */}
        <h1 id="hero-h" className="font-display text-[32px] text-balance sm:text-[40px]">
          จองโต๊ะก่อน<span className="whitespace-nowrap">ที่นั่ง<span className="text-accent">หมด</span></span>
        </h1>
        <p className="text-muted">เลือกวัน เวลา จำนวนคน แล้วกดเวลาที่ว่างบนการ์ดร้านได้เลย</p>
      </div>
      {/* มือถือ: 2 คอลัมน์ (เวลา + จำนวนคน อยู่แถวเดียวกัน) / desktop: แถวเดียว */}
      <form method="get" action="/" className="grid grid-cols-2 gap-4 lg:grid-cols-[1.6fr_1fr_0.8fr_0.8fr_auto] lg:items-end">
        <input type="hidden" name="sort" value={sort} />
        {cuisine && <input type="hidden" name="cuisine" value={cuisine} />}
        <label className={`${labelCls} col-span-2 lg:col-span-1`}>
          ร้านหรือเมนู
          <input name="q" type="search" defaultValue={q} placeholder="เช่น ซูชิ, ชาบู" className={inputCls} />
        </label>
        <label className={`${labelCls} col-span-2 lg:col-span-1`}>
          วันที่
          <input name="date" type="date" defaultValue={date} min={todayKey()} required className={inputCls} />
        </label>
        <label className={labelCls}>
          เวลาประมาณ
          <select name="time" defaultValue={time} className={inputCls}>
            {times.map((t) => <option key={t} value={t}>{t}</option>)}
          </select>
        </label>
        <label className={labelCls}>
          จำนวนคน
          <select name="party_size" defaultValue={party} className={inputCls}>
            {Array.from({ length: 20 }, (_, i) => i + 1).map((n) => <option key={n} value={n}>{n} คน</option>)}
          </select>
        </label>
        <button type="submit" className="btn btn-cta col-span-2 h-12 px-7 lg:col-span-1">
          <Search size={18} aria-hidden />
          ค้นหา
        </button>
      </form>
    </section>
  );
}
