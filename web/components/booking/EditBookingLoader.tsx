"use client";

import { signIn, useSession } from "next-auth/react";

import { Alert, Skeleton } from "@/components/bases/ui";
import { fmtTime } from "@/lib/format";
import type { Restaurant } from "@/lib/types";
import { useBooking } from "@/services/bookings";

import BookingPanel from "./BookingPanel";

/** โหมดแก้ไข (/restaurants/:id?edit=<bookingId>) — โหลดการจองเดิมแล้วเปิดแผงจองพร้อมค่าเดิม */
export default function EditBookingLoader({ bookingId, restaurant }: { bookingId: string; restaurant: Restaurant }) {
  const { status } = useSession();
  const booking = useBooking(bookingId);

  if (status === "unauthenticated") {
    void signIn("keycloak");
    return null;
  }
  if (booking.isLoading || status === "loading") return <Skeleton className="h-96" />;
  if (!booking.data) return <Alert tone="full" title="ไม่พบการจองที่จะแก้ไข" />;
  if (!booking.data.can_change) return <Alert tone="warn" title="การจองนี้แก้ไขไม่ได้แล้ว">เลยเวลาที่แก้ไขได้ หรือถูกยกเลิกไปแล้ว</Alert>;

  const b = booking.data;
  return (
    <BookingPanel restaurant={restaurant} editing={b} initialDate={b.business_date} initialTime={fmtTime(b.start_at)} initialParty={b.party_size} />
  );
}
