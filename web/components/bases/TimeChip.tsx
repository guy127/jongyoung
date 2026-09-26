import Link from "next/link";

import type { ChipState } from "@/lib/format";

// ปุ่มเวลา 3 สถานะ + "เลือกอยู่" (CLAUDE.md ข้อ 8.4): ว่าง = เขียว, เหลือน้อย = อำพัน, เต็ม = แดง, ปิด = เทา
// สถานะมีสัญลักษณ์และข้อความเสมอ ไม่ใช่สีอย่างเดียว
const styles: Record<ChipState | "selected" | "inRange", string> = {
  ok: "border border-ok-line bg-ok-weak text-ok",
  low: "border border-warn-line bg-warn-weak text-warn",
  full: "border border-full-line bg-full-weak text-full cursor-not-allowed",
  closed: "border border-border bg-chip text-soft cursor-not-allowed",
  selected: "border-2 border-sel bg-sel text-sel-text",
  inRange: "border-2 border-dashed border-sel bg-sel-weak text-text",
};

export function chipNote(state: ChipState, available: number) {
  switch (state) {
    case "ok":
      return "✓ ว่าง";
    case "low":
      return `! เหลือ ${available}`;
    case "full":
      return "✕ เต็ม";
    case "closed":
      return "ปิด";
  }
}

type Props = {
  time: string;
  note: string;
  state: ChipState;
  selected?: boolean;
  inRange?: boolean;
  dayLabel?: string; // "(เช้าวันที่ 11)" สำหรับช่วงหลังเที่ยงคืน
  href?: string; // ใช้บนการ์ดร้าน (ลิงก์ไปหน้าร้าน)
  onClick?: () => void; // ใช้ในแผงจอง
  className?: string;
};

export default function TimeChip({ time, note, state, selected, inRange, dayLabel, href, onClick, className = "" }: Props) {
  const disabled = state === "full" || state === "closed";
  const look = selected ? styles.selected : inRange ? styles.inRange : styles[state];
  // .time-chip = hover/กด เฉพาะปุ่มที่ยังกดได้และยังไม่ได้เลือก (ปุ่มเต็ม/ปิดต้องนิ่ง ไม่ชวนให้กด)
  const interactive = !disabled && !selected ? "time-chip cursor-pointer" : "";
  const cls = `flex min-h-14 min-w-14 flex-col items-center justify-center gap-0.5 rounded-xl px-1.5 leading-tight tabular ${look} ${interactive} ${className}`;
  const label = `${time}${dayLabel ? " " + dayLabel : ""} ${note.replace(/^[✓!✕] /, "")}`;
  const content = (
    <>
      {/* ขีดฆ่าเฉพาะเวลา — ขีดทับข้อความไทยแล้วสระ/วรรณยุกต์อ่านยาก */}
      <span className={`text-[15px] font-semibold ${state === "full" && !selected ? "line-through" : ""}`}>{time}</span>
      <span className="text-[13px] no-underline">{dayLabel ? dayLabel.replace(/[()]/g, "") : note}</span>
    </>
  );

  if (href && !disabled) {
    return <Link href={href} className={cls} aria-label={label}>{content}</Link>;
  }
  return (
    <button type="button" className={cls} disabled={disabled} aria-pressed={selected} aria-label={label} onClick={onClick}>
      {content}
    </button>
  );
}
