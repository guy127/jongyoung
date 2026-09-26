// component พื้นฐานที่ใช้ทั้งเว็บ — เขียนเองด้วย Tailwind ให้อธิบายได้ทุกบรรทัด
import { AlertTriangle, CheckCircle2, Info, XCircle } from "lucide-react";
import Link from "next/link";
import type { ButtonHTMLAttributes, ReactNode } from "react";

type Variant = "cta" | "outline" | "ghost" | "danger";

const variants: Record<Variant, string> = {
  cta: "cta font-semibold shadow-[0_10px_24px_rgba(197,22,46,.22)]", // ปุ่มหลัก 1 ปุ่มต่อหน้าจอ
  outline: "border border-border-strong bg-surface text-text",
  ghost: "text-text hover:bg-chip",
  danger: "border border-full-line bg-surface text-full",
};

export const buttonClass = (variant: Variant = "outline", extra = "") =>
  `inline-flex min-h-11 items-center justify-center gap-2 rounded-full px-4 text-[15px] disabled:cursor-not-allowed disabled:opacity-50 ${variants[variant]} ${extra}`;

export function Button({ variant = "outline", className = "", ...props }: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant }) {
  return <button type="button" className={buttonClass(variant, className)} {...props} />;
}

export function LinkButton({ href, variant = "outline", className = "", children }: { href: string; variant?: Variant; className?: string; children: ReactNode }) {
  return <Link href={href} className={buttonClass(variant, className)}>{children}</Link>;
}

type Tone = "ok" | "warn" | "full" | "info";
const tones: Record<Tone, { cls: string; Icon: typeof Info }> = {
  ok: { cls: "bg-ok-weak text-ok", Icon: CheckCircle2 },
  warn: { cls: "bg-warn-weak text-warn border border-warn-line", Icon: AlertTriangle },
  full: { cls: "bg-full-weak text-full border border-full-line", Icon: XCircle },
  info: { cls: "bg-sel-weak text-text border border-sel/40", Icon: Info }, // สีกลาง ใช้กับ DUPLICATE_BOOKING
};

/** กล่องแจ้งเตือน — มีไอคอน + ข้อความเสมอ ไม่สื่อความหมายด้วยสีอย่างเดียว */
export function Alert({ tone, title, children }: { tone: Tone; title: string; children?: ReactNode }) {
  const { cls, Icon } = tones[tone];
  return (
    <div role={tone === "full" ? "alert" : "status"} className={`flex gap-3 rounded-2xl p-4 ${cls}`}>
      <Icon size={20} className="mt-0.5 shrink-0" aria-hidden />
      <div className="flex flex-col gap-1">
        <p className="font-semibold">{title}</p>
        {children && <div className="text-sm leading-relaxed">{children}</div>}
      </div>
    </div>
  );
}

export function Skeleton({ className = "" }: { className?: string }) {
  return <div aria-hidden className={`animate-pulse rounded-xl bg-chip ${className}`} />;
}

export function EmptyState({ title, children, action }: { title: string; children?: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-border-strong p-8 text-center">
      <p className="text-lg font-semibold">{title}</p>
      {children && <div className="text-sm text-muted">{children}</div>}
      {action}
    </div>
  );
}

/** กล่องสำหรับ error ของการโหลดข้อมูล + ปุ่มลองใหม่ */
export function LoadError({ onRetry }: { onRetry: () => void }) {
  return (
    <div role="alert" className="flex items-center justify-between gap-3 rounded-2xl border border-border bg-surface p-4">
      <div>
        <p className="font-semibold">โหลดข้อมูลไม่สำเร็จ</p>
        <p className="text-sm text-muted">ตรวจสอบอินเทอร์เน็ตแล้วลองอีกครั้ง</p>
      </div>
      <Button onClick={onRetry}>ลองอีกครั้ง</Button>
    </div>
  );
}
