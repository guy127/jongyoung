import Image from "next/image";
import Link from "next/link";
import { notFound } from "next/navigation";

import Rating from "@/components/bases/Rating";
import BookingPanel from "@/components/booking/BookingPanel";
import EditBookingLoader from "@/components/booking/EditBookingLoader";
import ClosureBanner from "@/components/restaurant/ClosureBanner";
import ReviewsSection from "@/components/review/ReviewsSection";
import { serverGet } from "@/lib/api";
import { closedDaysLabel, defaultSearch, hoursLabel, mapHref } from "@/lib/format";
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
  const hours = `${hoursLabel(restaurant)}${restaurant.overnight ? " (ถึงเช้าวันถัดไป)" : ""}`;

  return (
    <div className="flex flex-col gap-6 sm:gap-8">
      <nav aria-label="เส้นทาง" className="-mb-2 flex gap-2 text-sm text-muted">
        <Link href="/" className="link no-underline hover:underline">ค้นหาร้าน</Link><span aria-hidden>/</span><span aria-current="page">{restaurant.name}</span>
      </nav>

      {/* ตัวอักษรอยู่บนแถบไล่เฉดทึบ (ไม่ใช่ blur ทับรูป) → contrast คงที่ไม่ว่ารูปจะสว่างแค่ไหน (ข้อ 8.2) */}
      <header className="relative flex h-64 items-end overflow-hidden rounded-3xl bg-chip sm:h-80">
        {restaurant.images[0] && <Image src={restaurant.images[0].url} alt={`รูปร้าน ${restaurant.name}`} fill priority sizes="1120px" className="object-cover" />}
        <div className="relative flex w-full flex-col gap-2 bg-linear-to-t from-black/80 via-black/55 to-transparent px-5 pb-5 pt-16 text-white sm:px-8 sm:pb-7">
          <h1 className="font-display text-[30px] text-balance sm:text-[40px]">{restaurant.name}</h1>
          <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-[15px] text-white/90 tabular">
            <Rating rating={restaurant.rating} /><span aria-hidden>·</span>{restaurant.cuisine}<span aria-hidden>·</span>{hours}
            {restaurant.closed_weekdays.length > 0 && <><span aria-hidden>·</span>{closedDaysLabel(restaurant.closed_weekdays)}</>}
            <span aria-hidden>·</span>{restaurant.seats} ที่นั่ง
          </p>
        </div>
      </header>

      <ClosureBanner restaurantId={restaurant.id} />

      <div className="grid grid-cols-1 gap-10 lg:grid-cols-[minmax(0,1fr)_400px] lg:items-start lg:gap-12">
        <div className="order-2 flex flex-col gap-12 lg:order-1">
          <section aria-labelledby="about-h" className="flex flex-col gap-3">
            <h2 id="about-h" className="text-[22px] font-semibold">เกี่ยวกับร้าน</h2>
            <p className="text-muted">{restaurant.description || "—"}</p>
            <p className="text-sm">
              ที่อยู่: {restaurant.address} ·{" "}
              <a className="link" target="_blank" rel="noreferrer"
                href={mapHref(restaurant.address, restaurant.map_url)}>เปิดแผนที่</a>
            </p>
            {restaurant.images.length > 1 && (
              <div className="mt-2 grid grid-cols-3 gap-3">
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
