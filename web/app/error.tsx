"use client";

import { useEffect } from "react";

import BrandMark from "@/components/bases/BrandMark";
import { Button, LinkButton } from "@/components/bases/ui";

/**
 * error boundary ระดับแอป — หน้าไหน throw ตอน render (เช่น API ล่ม) จะมาจบที่นี่แทนหน้าขาวของ Next
 * ไม่แสดง error.message/stack ให้ผู้ใช้ (อาจมี path ภายในหรือข้อมูลระบบ) — log ลง console พอ
 */
export default function AppError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <main className="grid min-h-screen place-items-center px-4">
      <div role="alert" className="flex w-full max-w-sm flex-col items-center gap-5 rounded-3xl border border-border bg-surface p-8 text-center">
        <span className="brand-mark grid size-12 place-items-center rounded-[14px]">
          <BrandMark size={36} />
        </span>
        <div className="flex flex-col gap-2">
          <h1 className="text-xl font-semibold">ระบบขัดข้องชั่วคราว</h1>
          <p className="text-sm text-muted">ลองใหม่อีกครั้ง ถ้ายังไม่ได้ ตรวจว่า API ทำงานอยู่หรือไม่</p>
        </div>
        <div className="flex w-full flex-col gap-2">
          {/* reset() = render ส่วนที่พังใหม่อีกครั้ง */}
          <Button variant="cta" className="w-full" onClick={reset}>ลองใหม่</Button>
          <LinkButton href="/" className="w-full">กลับหน้าแรก</LinkButton>
        </div>
      </div>
    </main>
  );
}
