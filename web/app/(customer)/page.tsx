import Link from "next/link";

import { ChoiceLink } from "@/components/bases/Choice";
import { PageLayout, Section } from "@/components/bases/layout";
import { EmptyState } from "@/components/bases/ui";
import RestaurantCard from "@/components/restaurant/RestaurantCard";
import SearchForm from "@/components/restaurant/SearchForm";
import { serverGet } from "@/lib/api";
import { defaultSearch, fmtShortDate, keyToDate } from "@/lib/format";
import type { ListItem, Page } from "@/lib/types";

type Search = { q?: string; cuisine?: string; sort?: string; page?: string; date?: string; time?: string; party_size?: string };

const sorts = [
  { key: "newest", label: "ใหม่ล่าสุด", hint: "ร้านที่เพิ่งเข้าร่วมขึ้นก่อน" },
  { key: "rating", label: "คะแนนสูงสุด", hint: "คะแนนถ่วงน้ำหนักด้วยจำนวนรีวิว — ร้านที่รีวิวน้อยจะไม่แซงร้านที่รีวิวเยอะ" },
  { key: "reviews", label: "รีวิวมากที่สุด", hint: "ร้านที่มีคนรีวิวมากที่สุดขึ้นก่อน" },
];

// server component: ค่าที่ค้นอยู่ใน URL ทั้งหมด (แชร์ลิงก์/refresh ได้) และไม่ต้องใช้ token
export default async function HomePage({ searchParams }: { searchParams: Promise<Search> }) {
  const sp = await searchParams;
  const def = defaultSearch();
  const search = {
    q: sp.q ?? "",
    cuisine: sp.cuisine ?? "",
    sort: sorts.some((s) => s.key === sp.sort) ? sp.sort! : "newest",
    date: sp.date ?? def.date,
    time: sp.time ?? def.time,
    party: Math.min(Math.max(Number(sp.party_size) || 2, 1), 30),
    page: Math.max(Number(sp.page) || 1, 1),
  };

  const qs = new URLSearchParams({
    q: search.q, cuisine: search.cuisine, sort: search.sort, page: String(search.page), limit: "9",
    date: search.date, time: search.time, party_size: String(search.party),
  });
  let data: Page<ListItem> | null = null;
  let failed = false;
  try {
    data = await serverGet<Page<ListItem>>(`/restaurants?${qs}`);
  } catch {
    failed = true;
  }

  // ลิงก์ที่คงค่าค้นหาเดิมไว้ เปลี่ยนแค่บางค่า (เรียงลำดับ/หน้า)
  const withParams = (patch: Record<string, string>) =>
    `/?${new URLSearchParams({ ...Object.fromEntries(qs), ...patch }).toString()}`;
  const totalPages = data ? Math.ceil(data.total / data.limit) : 0;
  const sortInfo = sorts.find((s) => s.key === search.sort)!;

  return (
    <PageLayout>
      <SearchForm q={search.q} date={search.date} time={search.time} party={search.party} sort={search.sort} cuisine={search.cuisine} />

      <Section
        id="results-h"
        title={`${data ? `${data.total} ร้าน` : "ร้านอาหาร"} · ${fmtShortDate(keyToDate(search.date))} ราว ${search.time} · ${search.party} คน`}
        description={sortInfo.hint}
        actions={
          <nav aria-label="เรียงลำดับ" className="-mx-4 flex gap-2 overflow-x-auto px-4 sm:mx-0 sm:px-0">
            {sorts.map((s) => (
              <ChoiceLink key={s.key} active={s.key === search.sort} href={withParams({ sort: s.key, page: "1" })}>{s.label}</ChoiceLink>
            ))}
          </nav>
        }
      >
        {failed && (
          <div role="alert" className="flex flex-col gap-1 rounded-2xl border border-border bg-surface p-5">
            <p className="font-semibold">โหลดรายการร้านไม่สำเร็จ</p>
            <Link href={withParams({})} className="link">ลองอีกครั้ง</Link>
          </div>
        )}

        {data && data.items.length === 0 && (
          <EmptyState title="ไม่พบร้านที่ตรงกับคำค้น">ลองค้นด้วยคำอื่น ลดจำนวนคน หรือเปลี่ยนวัน/เวลา</EmptyState>
        )}

        {data && data.items.length > 0 && (
          <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3 lg:gap-6">
            {data.items.map((r) => (
              <RestaurantCard key={r.id} restaurant={r} date={search.date} time={search.time} party={search.party} />
            ))}
          </div>
        )}

        {totalPages > 1 && (
          <nav aria-label="หน้า" className="flex justify-center gap-2 pt-4">
            {Array.from({ length: totalPages }, (_, i) => i + 1).map((p) => (
              <Link key={p} href={withParams({ page: String(p) })} aria-current={p === search.page ? "page" : undefined}
                className={`chip-btn grid size-11 place-items-center rounded-full ${p === search.page ? "border-sel font-semibold" : ""}`}>
                {p}
              </Link>
            ))}
          </nav>
        )}

        <p className="flex flex-wrap gap-4 text-[13px] text-muted">
          <span>ปุ่มเวลา:</span><span>✓ ว่าง</span><span className="text-warn">! เหลือน้อย</span><span className="text-soft">✕ เต็มสำหรับจำนวนคนนี้</span>
        </p>
      </Section>
    </PageLayout>
  );
}
