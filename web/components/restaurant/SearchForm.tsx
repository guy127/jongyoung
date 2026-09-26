import { todayKey } from "@/lib/format";

const times = Array.from({ length: 48 }, (_, i) => `${String(Math.floor(i / 2)).padStart(2, "0")}:${i % 2 ? "30" : "00"}`);
const inputCls = "h-12 w-full rounded-xl border border-border-strong bg-surface px-3 text-base text-text";

type Props = { q: string; date: string; time: string; party: number; sort: string; cuisine: string };

/** แถบค้นหา (งานหลักของหน้าแรก) — form แบบ GET ธรรมดา ส่งค่าผ่าน URL ไม่ต้องใช้ JavaScript */
export default function SearchForm({ q, date, time, party, sort, cuisine }: Props) {
  return (
    <section aria-labelledby="hero-h" className="glass flex flex-col gap-4 rounded-3xl border border-border p-5 sm:p-7">
      <div>
        <h1 id="hero-h" className="font-display text-[36px] sm:text-[40px]">
          จองโต๊ะก่อนที่นั่ง<span className="text-accent">หมด</span>
        </h1>
        <p className="text-muted">เลือกวัน เวลา จำนวนคน แล้วกดเวลาที่ว่างบนการ์ดร้านได้เลย</p>
      </div>
      <form method="get" action="/" className="grid gap-3 sm:grid-cols-[1.6fr_1fr_0.8fr_0.8fr_auto] sm:items-end">
        <input type="hidden" name="sort" value={sort} />
        {cuisine && <input type="hidden" name="cuisine" value={cuisine} />}
        <label className="flex flex-col gap-1.5 text-sm font-medium">
          ร้านหรือเมนู
          <input name="q" type="search" defaultValue={q} placeholder="เช่น ซูชิ, ชาบู" className={inputCls} />
        </label>
        <label className="flex flex-col gap-1.5 text-sm font-medium">
          วันที่
          <input name="date" type="date" defaultValue={date} min={todayKey()} required className={inputCls} />
        </label>
        <label className="flex flex-col gap-1.5 text-sm font-medium">
          เวลาประมาณ
          <select name="time" defaultValue={time} className={inputCls}>
            {times.map((t) => <option key={t} value={t}>{t}</option>)}
          </select>
        </label>
        <label className="flex flex-col gap-1.5 text-sm font-medium">
          จำนวนคน
          <select name="party_size" defaultValue={party} className={inputCls}>
            {Array.from({ length: 20 }, (_, i) => i + 1).map((n) => <option key={n} value={n}>{n} คน</option>)}
          </select>
        </label>
        <button type="submit" className="cta h-12 rounded-full px-7 font-semibold">ค้นหา</button>
      </form>
    </section>
  );
}
