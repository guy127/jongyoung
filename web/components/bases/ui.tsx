// component พื้นฐานที่ใช้ทั้งเว็บ — เขียนเองด้วย Tailwind ให้อธิบายได้ทุกบรรทัด
import { AlertTriangle, CheckCircle2, Info, Loader2, XCircle } from "lucide-react";
import Link from "next/link";
import type { ButtonHTMLAttributes, ReactNode } from "react";

// cta = ปุ่มหลัก ใช้ได้ 1 ปุ่มต่อหน้าจอ — hover/กด/disabled ของทุกแบบอยู่ใน globals.css (.btn-*)
type Variant = "cta" | "outline" | "ghost" | "danger";

export const buttonClass = (variant: Variant = "outline", extra = "") => `btn btn-${variant} ${extra}`;

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant; loading?: boolean };

/** loading = กำลังรอ API: แสดง spinner แทนไอคอน, กดซ้ำไม่ได้, screen reader รู้ผ่าน aria-busy */
export function Button({ variant = "outline", className = "", loading = false, disabled, children, ...props }: ButtonProps) {
  return (
    <button type="button" className={buttonClass(variant, className)} disabled={disabled || loading} aria-busy={loading || undefined} {...props}>
      {loading && <Loader2 size={18} className="animate-spin motion-reduce:animate-none" aria-hidden />}
      {children}
    </button>
  );
}

export function LinkButton({ href, variant = "outline", className = "", children }: { href: string; variant?: Variant; className?: string; children: ReactNode }) {
  return <Link href={href} className={buttonClass(variant, className)}>{children}</Link>;
}

type Tone = "ok" | "warn" | "full" | "info";
const tones: Record<Tone, { cls: string; Icon: typeof Info }> = {
  ok: { cls: "bg-ok-weak text-ok border border-ok-line", Icon: CheckCircle2 },
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
