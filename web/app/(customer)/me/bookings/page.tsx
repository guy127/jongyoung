"use client";

import { Clock, Info } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { ChoiceButton } from "@/components/bases/Choice";
import { PageHeader } from "@/components/bases/layout";
import { Alert, Button, EmptyState, LinkButton, LoadError, Skeleton } from "@/components/bases/ui";
import { apiError } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { fmtRange, fmtShortDate, fmtTime, keyToDate } from "@/lib/format";
import type { Booking } from "@/lib/types";
import { useCancelBooking, useMyBookings } from "@/services/bookings";

const tabs = [
  { key: "upcoming", label: "กำลังจะถึง" },
  { key: "past", label: "ผ่านมาแล้ว" },
  { key: "cancelled", label: "ยกเลิกแล้ว" },
] as const;

export default function MyBookingsPage() {
  const [tab, setTab] = useState<(typeof tabs)[number]["key"]>("upcoming");
  const list = useMyBookings(tab);

  return (
    // หน้านี้มีเนื้อหาก้อนเดียว (หัว + รายการ) จึงใช้ระยะ 32px แทนระยะระหว่าง section
    <div className="mx-auto flex w-full max-w-[880px] flex-col gap-6 sm:gap-8">
      <PageHeader
        title="การจองของฉัน"
        description="แก้ไขหรือยกเลิกได้จากหน้านี้ ภายในเวลาที่แต่ละร้านกำหนด"
        actions={
          <div role="tablist" aria-label="สถานะการจอง" className="-mx-4 flex gap-2 overflow-x-auto px-4 sm:mx-0 sm:px-0">
            {tabs.map((t) => (
              <ChoiceButton key={t.key} role="tab" active={t.key === tab} onClick={() => setTab(t.key)}>{t.label}</ChoiceButton>
            ))}
          </div>
        }
      />

      <div className="flex flex-col gap-5">
        {list.isLoading && [0, 1].map((i) => <Skeleton key={i} className="h-40" />)}
        {list.isError && <LoadError onRetry={() => list.refetch()} />}
        {list.data?.length === 0 && (
          <EmptyState title={tab === "upcoming" ? "ยังไม่มีการจองที่กำลังจะถึง" : "ยังไม่มีรายการ"} action={<LinkButton href="/">ค้นหาร้าน</LinkButton>}>
            หาร้านแล้วกดเวลาที่ว่างบนการ์ดได้เลย
          </EmptyState>
        )}
        {list.data?.map((b) => <BookingCard key={b.id} b={b} past={tab === "past"} />)}
      </div>
    </div>
  );
}

function BookingCard({ b, past }: { b: Booking; past: boolean }) {
  const cancel = useCancelBooking();
  const [confirming, setConfirming] = useState(false);
  const cancelled = b.status === "cancelled";
  const active = b.status === "active" && !past;
  const date = keyToDate(b.business_date);

  return (
    <article className="overflow-hidden rounded-2xl border border-border bg-surface">
      <div className="grid grid-cols-[auto_1fr] items-center gap-x-4 gap-y-5 p-5 sm:grid-cols-[auto_1fr_auto] sm:gap-x-5 sm:p-6">
        <div className={`flex h-20 w-18 flex-col items-center justify-center rounded-xl text-center ${cancelled ? "bg-chip text-soft" : "bg-full-weak text-full"}`}>
          <span className="text-[13px]">{fmtShortDate(date).split(" ")[0]}</span>
          <span className="text-2xl font-semibold leading-none tabular">{Number(b.business_date.slice(8))}</span>
          <span className="text-[13px]">{fmtShortDate(date).split(" ").slice(2).join(" ")}</span>
        </div>
        <div className="flex flex-col gap-1">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <Link href={`/bookings/${b.id}`} className="text-[19px] font-semibold">{b.restaurant.name}</Link>
            <span className={`rounded-full px-2.5 text-[13px] font-medium ${cancelled ? "bg-full-weak text-full" : past ? "bg-chip text-muted" : "bg-ok-weak text-ok"}`}>
              {cancelled ? "✕ ยกเลิกแล้ว" : past ? "ไปแล้ว" : "✓ ยืนยันแล้ว"}
            </span>
          </div>
          <span className="tabular">{fmtRange(b.start_at, b.end_at, b.business_date)} · {b.party_size} คน</span>
          <span className="text-[13px] text-muted tabular">เลขที่จอง {b.code}</span>
        </div>
        <div className="col-span-2 flex gap-3 sm:col-span-1 [&>*]:flex-1 sm:[&>*]:flex-none">
          {active && (
            <>
              {b.can_change
                ? <LinkButton href={`/restaurants/${b.restaurant.id}?edit=${b.id}`}>แก้ไข</LinkButton>
                : <Button disabled>แก้ไข</Button>}
              <Button variant="danger" disabled={!b.can_change} onClick={() => setConfirming(true)}>ยกเลิกการจอง</Button>
            </>
          )}
          {past && <LinkButton href={`/restaurants/${b.restaurant.id}#rev-h`}>เขียนรีวิว</LinkButton>}
        </div>
      </div>

      {active && (
        // ปุ่มที่กดไม่ได้ต้องบอกเหตุผล ไม่ใช่หายไปเฉย ๆ
        <p className={`flex items-start gap-2 border-t border-border px-5 py-4 text-sm sm:px-6 ${b.can_change ? "text-muted" : "bg-chip"}`}>
          {b.can_change ? <Clock size={16} className="mt-1 shrink-0" aria-hidden /> : <Info size={16} className="mt-1 shrink-0" aria-hidden />}
          {b.can_change
            ? `แก้ไขหรือยกเลิกได้ถึง ${fmtShortDate(b.cancel_until)} ${fmtTime(b.cancel_until)} น.`
            : `เลยเวลายกเลิกแล้ว (ได้ถึง ${fmtShortDate(b.cancel_until)} ${fmtTime(b.cancel_until)} น.) หากไปไม่ได้โปรดแจ้งร้านโดยตรง`}
        </p>
      )}

      {confirming && (
        <div role="alertdialog" aria-label="ยืนยันการยกเลิก" className="mx-5 mb-5 flex flex-wrap items-center gap-3 rounded-xl border border-full-line bg-full-weak p-4 sm:mx-6 sm:mb-6">
          <div className="flex-1">
            <p className="font-semibold">ยกเลิกการจอง {b.restaurant.name}?</p>
            <p className="text-sm text-muted">ที่นั่ง {b.party_size} ที่จะเปิดให้คนอื่นจองทันที และกู้คืนไม่ได้</p>
          </div>
          <Button onClick={() => setConfirming(false)}>ไม่ยกเลิก</Button>
          <Button variant="danger" loading={cancel.isPending} onClick={() => cancel.mutate(b.id, { onSuccess: () => setConfirming(false) })}>
            {cancel.isPending ? "กำลังยกเลิก…" : "ยืนยันยกเลิก"}
          </Button>
          {cancel.isError && <div className="w-full"><Alert tone="full" title={errorMessage(apiError(cancel.error))} /></div>}
        </div>
      )}
    </article>
  );
}
