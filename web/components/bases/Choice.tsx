// ปุ่ม "เลือกหนึ่งค่า" (วัน, ระยะเวลา, แท็บ, การเรียงลำดับ) — เขียนครั้งเดียวใช้ทั้งระบบ
// ค่าที่เลือกบอกด้วย aria-pressed (ปุ่ม) หรือ aria-current (ลิงก์) ไม่ใช่สีอย่างเดียว
import Link from "next/link";
import type { ReactNode } from "react";

type Tone = "dark" | "sel"; // dark = แท็บ/ตัวกรอง, sel = ค่าที่ผู้ใช้เลือกในแผงจอง (สี --sel)

export const choiceClass = (active: boolean, tone: Tone = "dark", extra = "") =>
  `chip-btn inline-flex min-h-11 shrink-0 items-center justify-center rounded-full px-4 text-sm ${
    active ? (tone === "sel" ? "border-sel bg-sel text-sel-text" : "border-text bg-text text-surface") : ""
  } ${extra}`;

export function ChoiceButton({ active, tone, onClick, children, className, role }: {
  active: boolean;
  tone?: Tone;
  onClick: () => void;
  children: ReactNode;
  className?: string;
  role?: "tab";
}) {
  const state = role === "tab" ? { "aria-selected": active } : { "aria-pressed": active };
  return (
    <button type="button" role={role} {...state} onClick={onClick} className={choiceClass(active, tone, className)}>
      {children}
    </button>
  );
}

export function ChoiceLink({ active, href, children }: { active: boolean; href: string; children: ReactNode }) {
  return (
    <Link href={href} aria-current={active ? "true" : undefined} className={choiceClass(active)}>
      {children}
    </Link>
  );
}
