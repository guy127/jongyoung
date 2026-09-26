"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import BrandMark from "@/components/bases/BrandMark";
import ThemeToggle from "@/components/bases/ThemeToggle";
import { useMe } from "@/services/me";

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
      <nav aria-label="เมนูเจ้าของร้าน" className="mx-auto flex h-14 max-w-[1216px] items-center gap-4 px-4 text-sm">
        {/* ฝั่ง owner ไม่มี gradient (ข้อ 8.5) — โลโก้เป็นเส้นขาวล้วนบนแถบทึบ */}
        <Link href="/owner/restaurants" className="flex items-center gap-2 text-base font-semibold">
          <BrandMark size={24} />
          จองยัง
        </Link>
        <span className="rounded-md bg-white/15 px-2 py-0.5 text-[13px]">โหมดเจ้าของร้าน</span>
        <div className="flex flex-1 gap-1">
          {item("/owner/restaurants", "ร้านของฉัน", pathname === "/owner/restaurants")}
          {item("/owner/bookings", "การจองรายวัน", pathname.startsWith("/owner/bookings"))}
        </div>
        <div role="group" aria-label="โหมดการใช้งาน" className="flex overflow-hidden rounded-lg border border-white/30">
          <Link href="/" className="px-3 py-1.5 text-white/85 transition-colors hover:bg-white/10 hover:text-white">ลูกค้า</Link>
          <span aria-current="true" className="bg-white px-3 py-1.5 font-semibold text-[var(--owner-bar)]">เจ้าของร้าน</span>
        </div>
        <ThemeToggle className="border-white/30 text-white transition-colors hover:bg-white/10" />
        <span className="hidden md:inline">{me?.display_name}</span>
        {/* <a> ธรรมดาโดยตั้งใจ — <Link> จะ prefetch route handler logout (ดู Navbar) */}
        {/* eslint-disable-next-line @next/next/no-html-link-for-pages */}
        <a href="/api/auth/logout" className="rounded-md px-2 py-1 text-white/80 transition-colors hover:bg-white/10 hover:text-white">ออกจากระบบ</a>
      </nav>
    </header>
  );
}
