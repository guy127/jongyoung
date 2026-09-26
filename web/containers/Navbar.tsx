"use client";

import { CalendarCheck, LogIn, LogOut, Search } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { signIn, useSession } from "next-auth/react";

import BrandMark from "@/components/bases/BrandMark";
import ThemeToggle from "@/components/bases/ThemeToggle";
import { buttonClass } from "@/components/bases/ui";
import { useMe } from "@/services/me";

/**
 * navbar ฝั่งลูกค้า — กระจกลอย (1 ใน 3 ที่ที่ใช้ glass) + ตัวสลับโหมดเมื่อผู้ใช้มีร้านของตัวเอง
 * มือถือ: เมนูเหลือแค่ไอคอน (ข้อความอยู่ใน sr-only ให้ screen reader) → อยู่แถวเดียวที่ 375px
 */
export default function Navbar() {
  const pathname = usePathname();
  const { status } = useSession();
  const { data: me } = useMe();
  const isOwner = (me?.restaurants.length ?? 0) > 0;

  const link = (href: string, label: string, Icon: typeof Search) => {
    const active = href === "/" ? pathname === "/" : pathname.startsWith(href);
    return (
      <Link href={href} aria-current={active ? "page" : undefined}
        className={`nav-link inline-flex min-h-11 items-center gap-2 rounded-full px-3 text-[15px] ${active ? "bg-chip font-medium text-text" : "text-muted"}`}>
        <Icon size={18} aria-hidden className="sm:hidden" />
        <span className="sr-only sm:not-sr-only">{label}</span>
      </Link>
    );
  };

  return (
    <header className="sticky top-3 z-20 mx-auto mt-3 w-[min(1120px,calc(100%-2rem))]">
      <nav aria-label="เมนูหลัก" className="glass flex items-center gap-1 rounded-full border border-border py-1.5 pl-2 pr-1.5 shadow-[0_16px_40px_rgba(120,36,36,.10)] sm:gap-2 sm:pl-3">
        <Link href="/" className="group mr-1 flex items-center gap-2 font-display text-xl">
          <span className="brand-mark grid size-9 place-items-center rounded-[10px] transition-transform duration-300 group-hover:-rotate-8 motion-reduce:transition-none">
            <BrandMark size={27} />
          </span>
          <span className="hidden sm:inline">จองยัง</span>
        </Link>
        <div className="flex flex-1 items-center gap-1">
          {link("/", "ค้นหาร้าน", Search)}
          {status === "authenticated" && link("/me/bookings", "การจองของฉัน", CalendarCheck)}
        </div>

        {isOwner && (
          <div role="group" aria-label="โหมดการใช้งาน" className="hidden rounded-full border border-border bg-chip p-0.5 md:flex">
            <span aria-current="true" className="rounded-full bg-surface px-3 py-1.5 text-sm font-semibold">ลูกค้า</span>
            <Link href="/owner/restaurants" className="nav-link rounded-full px-3 py-1.5 text-sm text-muted">เจ้าของร้าน</Link>
          </div>
        )}
        <ThemeToggle className="nav-link border-border" />
        {status === "authenticated" ? (
          // ใช้ <a> ธรรมดาโดยตั้งใจ: <Link> จะ prefetch route handler นี้ = อาจ logout ตั้งแต่ render ลิงก์
          // eslint-disable-next-line @next/next/no-html-link-for-pages
          <a href="/api/auth/logout" className="nav-link inline-flex min-h-11 items-center gap-2 rounded-full px-3 text-sm text-muted" title={me?.display_name}>
            <LogOut size={18} aria-hidden className="sm:hidden" />
            <span className="sr-only sm:not-sr-only">ออกจากระบบ</span>
          </a>
        ) : (
          <button type="button" className={buttonClass("outline", "px-3 sm:px-4")} onClick={() => signIn("keycloak")} disabled={status === "loading"}>
            <LogIn size={17} aria-hidden />
            เข้าสู่ระบบ
          </button>
        )}
      </nav>
    </header>
  );
}
