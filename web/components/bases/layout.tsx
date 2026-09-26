// โครงหน้าและระยะห่างมาตรฐาน — ทุกหน้าใช้ชุดนี้ ระยะจึงเท่ากันทั้งระบบ (scale 4 8 12 16 24 32 48 ข้อ 8.3)
//   ระหว่าง section ของหน้า      48px (มือถือ 40px)
//   หัวข้อ section → เนื้อหา     20px
//   ภายในการ์ด                  24px (มือถือ 20px)
//   ป้ายกำกับ → ช่องกรอก         8px   ช่องกรอก → ช่องกรอก 20px
import type { ReactNode } from "react";

export function PageLayout({ children, narrow = false }: { children: ReactNode; narrow?: boolean }) {
  return <div className={`mx-auto flex w-full flex-col gap-10 sm:gap-12 ${narrow ? "max-w-[880px]" : ""}`}>{children}</div>;
}

type HeaderProps = { title: string; description?: ReactNode; actions?: ReactNode };

/** หัวหน้า (h1) + คำอธิบาย + ปุ่มด้านขวา */
export function PageHeader({ title, description, actions }: HeaderProps) {
  return (
    <header className="flex flex-wrap items-end justify-between gap-x-6 gap-y-4">
      <div className="flex max-w-2xl flex-col gap-2">
        <h1 className="text-[28px] font-semibold text-balance sm:text-[32px]">{title}</h1>
        {description && <p className="text-muted">{description}</p>}
      </div>
      {actions}
    </header>
  );
}

/** section ของหน้า: หัวข้อ h2 + เนื้อหา — id ของหัวข้อใช้ผูก aria-labelledby */
export function Section({ id, title, description, actions, children }: HeaderProps & { id: string; children: ReactNode }) {
  return (
    <section aria-labelledby={id} className="flex flex-col gap-5">
      <div className="flex flex-wrap items-end justify-between gap-x-6 gap-y-3">
        <div className="flex flex-col gap-1">
          <h2 id={id} className="text-[22px] font-semibold text-balance">{title}</h2>
          {description && <p className="text-sm text-muted">{description}</p>}
        </div>
        {actions}
      </div>
      {children}
    </section>
  );
}

/** การ์ดพื้นทึบ (ไม่ใช้ glass — ข้อ 8.0) */
export function Card({ children, className = "", as: Tag = "div" }: { children: ReactNode; className?: string; as?: "div" | "section" | "article" | "form" }) {
  return <Tag className={`rounded-2xl border border-border bg-surface p-5 sm:p-6 ${className}`}>{children}</Tag>;
}

/** ป้ายกำกับ + ช่องกรอก/ตัวเลือก (ใช้ในฟอร์มและแผงจอง) */
export function Field({ label, hint, children, id }: { label: string; hint?: ReactNode; children: ReactNode; id?: string }) {
  return (
    <div className="flex flex-col gap-2">
      <span id={id} className="text-sm font-medium">{label}</span>
      {children}
      {hint && <span className="text-[13px] text-soft">{hint}</span>}
    </div>
  );
}
