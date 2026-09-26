"use client";

import { Moon, Sun } from "lucide-react";

/**
 * สลับ Day (ค่าเริ่มต้น) / Night — เก็บใน localStorage ของ browser นี้
 * ไม่มี state ใน React: ธีมอยู่ที่ <html data-theme> อย่างเดียว แล้วให้ CSS เลือกไอคอน/ข้อความที่จะโชว์
 * (ถ้าเก็บใน state ด้วยจะมีค่าสองที่ต้องคอย sync และ server render ไม่รู้ธีมของ browser)
 */
export default function ThemeToggle({ className = "" }: { className?: string }) {
  const toggle = () => {
    const html = document.documentElement;
    const night = html.dataset.theme !== "night";
    if (night) html.dataset.theme = "night";
    else delete html.dataset.theme;
    try {
      localStorage.setItem("jy-theme", night ? "night" : "day");
    } catch {
      // private mode ใช้ localStorage ไม่ได้ — สลับธีมได้แค่หน้านี้ ไม่เป็นไร
    }
  };

  return (
    <button type="button" onClick={toggle} aria-label="สลับโหมดสว่าง/มืด"
      className={`inline-flex h-10 items-center gap-1.5 rounded-full border px-3 text-sm ${className}`}>
      <span className="theme-day inline-flex items-center gap-1.5"><Sun size={16} aria-hidden />Day</span>
      <span className="theme-night items-center gap-1.5"><Moon size={16} aria-hidden />Night</span>
    </button>
  );
}
