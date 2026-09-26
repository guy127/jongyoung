"use client";

import { LogOut } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

import BrandMark from "@/components/bases/BrandMark";
import ThemeToggle from "@/components/bases/ThemeToggle";
import { useMe } from "@/services/me";
import NotificationBell from "@/containers/NotificationBell";

/** navbar โหมดเจ้าของร้าน — แถบทึบสีเข้ม คนละภาษาการออกแบบกับฝั่งลูกค้า ให้รู้ทันทีว่าอยู่โหมดไหน */
export default function OwnerNavbar() {
  const pathname = usePathname();
  const { data: me } = useMe();
  const item = (href: string, label: string, active: boolean) => (
    <Link href={href} aria-current={active ? "page" : undefined}
      className={`rounded-md px-3 py-2 transition-colors ${active ? "bg-white/15 text-white" : "text-white/80 hover:bg-white/10 hover:text-white"}`}>
      {label}
    </Link>
  );
  return (
    <header className="bg-[var(--owner-bar)] text-white">
      {/* มือถือ: แบ่งสองแถว — แถวบนโลโก้ + ตัวสลับโหมด + ปุ่ม, แถวล่างเมนู (order-last) ไม่ให้ข้อความหักบรรทัดหรือล้นจอ */}
      <nav aria-label="เมนูเจ้าของร้าน" className="mx-auto flex max-w-[1216px] flex-wrap items-center gap-x-1 gap-y-1 whitespace-nowrap px-4 py-2 text-sm md:h-14 md:flex-nowrap md:gap-4 md:py-0">
        {/* ฝั่ง owner ไม่มี gradient (ข้อ 8.5) — โลโก้เป็นเส้นขาวล้วนบนแถบทึบ */}
        <Link href="/owner/restaurants" className="flex items-center gap-2 text-base font-semibold">
          <BrandMark size={24} />
          <span className="hidden sm:inline">จองยัง</span>
        </Link>
        {/* มือถือซ่อนป้าย — ตัวสลับโหมดบอกอยู่แล้วว่าอยู่โหมดเจ้าของร้าน */}
        <span className="hidden rounded-md bg-white/15 px-2 py-0.5 text-[13px] md:inline">โหมดเจ้าของร้าน</span>
        <div className="order-last flex w-full gap-1 md:order-0 md:w-auto md:flex-1">
          {item("/owner/restaurants", "ร้านของฉัน", pathname === "/owner/restaurants")}
          {item("/owner/bookings", "การจองรายวัน", pathname.startsWith("/owner/bookings"))}
          {item("/owner/closures", "ปิดร้านชั่วคราว", pathname.startsWith("/owner/closures"))}
        </div>
        <div role="group" aria-label="โหมดการใช้งาน" className="ml-auto flex overflow-hidden rounded-lg border border-white/30 md:ml-0">
          <Link href="/" className="px-3 py-1.5 text-white/85 transition-colors hover:bg-white/10 hover:text-white">ลูกค้า</Link>
          <span aria-current="true" className="bg-white px-3 py-1.5 font-semibold text-[var(--owner-bar)]">เจ้าของร้าน</span>
        </div>
        <NotificationBell tone="dark" />
        <ThemeToggle className="border-white/30 text-white transition-colors hover:bg-white/10" />
        <span className="hidden md:inline">{me?.display_name}</span>
        {/* <a> ธรรมดาโดยตั้งใจ — <Link> จะ prefetch route handler logout (ดู Navbar) */}
        {/* eslint-disable-next-line @next/next/no-html-link-for-pages */}
        <a href="/api/auth/logout" className="inline-flex min-h-11 min-w-11 items-center justify-center rounded-md px-2 text-white/80 transition-colors hover:bg-white/10 hover:text-white md:min-h-0 md:py-1">
          <LogOut size={18} aria-hidden className="md:hidden" />
          <span className="sr-only md:not-sr-only">ออกจากระบบ</span>
        </a>
      </nav>
    </header>
  );
}
