import Image from "next/image";
import Link from "next/link";
import { notFound } from "next/navigation";

import Rating from "@/components/bases/Rating";
import BookingPanel from "@/components/booking/BookingPanel";
import EditBookingLoader from "@/components/booking/EditBookingLoader";
import ReviewsSection from "@/components/review/ReviewsSection";
import { serverGet } from "@/lib/api";
import { defaultSearch } from "@/lib/format";
import type { Page, Restaurant, Review } from "@/lib/types";

type Search = { date?: string; time?: string; party_size?: string; edit?: string };

// server component: ข้อมูลร้าน + รีวิวหน้าแรก (สาธารณะ) / แผงจองและรีวิวเป็น client island
export default async function RestaurantPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<Search> }) {
  const { id } = await params;
  const sp = await searchParams;
  const [restaurant, reviews] = await Promise.all([
    serverGet<Restaurant>(`/restaurants/${id}`),
    serverGet<Page<Review>>(`/restaurants/${id}/reviews?limit=5`),
  ]);
  if (!restaurant) notFound();

  const def = defaultSearch();
  const date = sp.date ?? def.date;
  const party = Math.min(Math.max(Number(sp.party_size) || 2, 1), restaurant.seats);
  const hours = restaurant.open_24h ? "เปิด 24 ชม." : `${restaurant.open_time}–${restaurant.close_time}${restaurant.overnight ? " (ถึงเช้าวันถัดไป)" : ""}`;

  return (
    <div className="flex flex-col gap-6">
      <nav aria-label="เส้นทาง" className="flex gap-2 text-sm text-muted">
        <Link href="/" className="link no-underline hover:underline">ค้นหาร้าน</Link><span aria-hidden>/</span><span aria-current="page">{restaurant.name}</span>
      </nav>

      <header className="relative h-64 overflow-hidden rounded-3xl bg-chip sm:h-80">
        {restaurant.images[0] && <Image src={restaurant.images[0].url} alt={`รูปร้าน ${restaurant.name}`} fill priority sizes="1120px" className="object-cover" />}
        <div className="absolute inset-x-4 bottom-4 flex flex-col gap-1 rounded-2xl bg-black/55 p-4 text-white backdrop-blur-md">
          <h1 className="font-display text-[32px] sm:text-[36px]">{restaurant.name}</h1>
          <p className="flex flex-wrap items-center gap-1.5 text-[15px] tabular">
            <Rating rating={restaurant.rating} /> · {restaurant.cuisine} · {hours} · {restaurant.seats} ที่นั่ง
          </p>
        </div>
      </header>

      <div className="grid gap-7 lg:grid-cols-[1fr_380px] lg:items-start">
        <div className="order-2 flex flex-col gap-8 lg:order-1">
          <section aria-labelledby="about-h" className="flex flex-col gap-2">
            <h2 id="about-h" className="text-xl font-semibold">เกี่ยวกับร้าน</h2>
            <p className="text-muted">{restaurant.description || "—"}</p>
            <p className="text-sm">
              ที่อยู่: {restaurant.address} ·{" "}
              <a className="link" target="_blank" rel="noreferrer"
                href={`https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(restaurant.address)}`}>เปิดแผนที่</a>
            </p>
            {restaurant.images.length > 1 && (
              <div className="mt-2 grid grid-cols-3 gap-2">
                {restaurant.images.slice(1, 4).map((img) => (
                  <div key={img.id} className="relative aspect-[4/3] overflow-hidden rounded-xl bg-chip">
                    <Image src={img.url} alt={`รูปเพิ่มเติมของ ${restaurant.name}`} fill sizes="240px" className="object-cover" />
                  </div>
                ))}
              </div>
            )}
          </section>
          <ReviewsSection restaurant={restaurant} initial={reviews ?? undefined} />
        </div>

        <div className="order-1 lg:order-2">
          {sp.edit ? (
            <EditBookingLoader bookingId={sp.edit} restaurant={restaurant} />
          ) : (
            <BookingPanel restaurant={restaurant} initialDate={date} initialTime={sp.time} initialParty={party} />
          )}
        </div>
      </div>
    </div>
  );
}
