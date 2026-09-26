"use client";

import { Bell } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { fmtShortDate, fmtTime } from "@/lib/format";
import { bellLabel, notificationHref, notificationText } from "@/lib/notifications";
import type { Notification } from "@/lib/types";
import { useMarkAllRead, useMarkRead, useNotifications } from "@/services/notifications";

/**
 * กระดิ่งแจ้งเตือน (ทั้งโหมดลูกค้าและเจ้าของร้าน) — tone="dark" ใช้บนแถบทึบของ owner
 * ตัวเลขยังไม่อ่านมีใน aria-label ด้วย (ไม่สื่อด้วยสี/ตัวเลขลอย ๆ อย่างเดียว)
 */
export default function NotificationBell({ tone = "light" }: { tone?: "light" | "dark" }) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const list = useNotifications();
  const markRead = useMarkRead();
  const markAll = useMarkAllRead();
  if (!list.data) return null; // ยังไม่ login หรือกำลังโหลดครั้งแรก

  const unread = list.data.unread_count;
  const go = (n: Notification) => {
    setOpen(false);
    if (!n.read_at) markRead.mutate(n.id);
    router.push(notificationHref(n));
  };

  return (
    <div className="relative" onKeyDown={(e) => e.key === "Escape" && setOpen(false)}>
      <button type="button" aria-label={bellLabel(unread)} aria-expanded={open} aria-haspopup="true" onClick={() => setOpen(!open)}
        className={`relative grid size-11 place-items-center rounded-full ${tone === "dark" ? "text-white/85 transition-colors hover:bg-white/10" : "nav-link text-muted"}`}>
        <Bell size={19} aria-hidden />
        {unread > 0 && (
          <span aria-hidden className="absolute right-0.5 top-0.5 grid h-5 min-w-5 place-items-center rounded-full bg-full px-1 text-[12px] font-semibold text-white tabular">
            {unread > 99 ? "99+" : unread}
          </span>
        )}
      </button>
      {open && (
        <div className="absolute right-0 top-12 z-30 w-[min(360px,calc(100vw-2rem))] overflow-hidden rounded-2xl border border-border bg-surface text-sm text-text shadow-lg">
          <div className="flex items-center justify-between border-b border-border px-4 py-2.5">
            <span className="font-semibold">การแจ้งเตือน</span>
            {unread > 0 && <button type="button" className="link" onClick={() => markAll.mutate()}>อ่านทั้งหมด</button>}
          </div>
          {list.data.items.length === 0 ? (
            <p className="p-4 text-muted">ยังไม่มีการแจ้งเตือน</p>
          ) : (
            <ul className="max-h-96 overflow-y-auto">
              {list.data.items.map((n) => (
                <li key={n.id} className="border-b border-border last:border-0">
                  <button type="button" onClick={() => go(n)} className={`flex w-full gap-2.5 px-4 py-3 text-left hover:bg-chip ${n.read_at ? "text-muted" : ""}`}>
                    <span aria-hidden className={`mt-1.5 size-2 shrink-0 rounded-full ${n.read_at ? "" : "bg-sel"}`} />
                    <span className="flex flex-col gap-0.5">
                      <span>{!n.read_at && <span className="sr-only">ยังไม่อ่าน: </span>}{notificationText(n)}</span>
                      <span className="text-[13px] text-soft tabular">{fmtShortDate(n.created_at)} {fmtTime(n.created_at)}</span>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
