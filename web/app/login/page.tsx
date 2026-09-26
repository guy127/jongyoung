"use client";

import { Loader2 } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { signIn } from "next-auth/react";
import { Suspense, useEffect } from "react";

import BrandMark from "@/components/bases/BrandMark";
import { Button } from "@/components/bases/ui";

/**
 * หน้าที่ proxy.ts / next-auth พามาเมื่อยังไม่ login — ส่งต่อไป Keycloak ทันที (ไม่มีหน้ากดซ้ำคั่นกลาง)
 * ถ้ากลับมาพร้อม ?error= (Keycloak ล่ม, ผู้ใช้กดยกเลิก) จะไม่ส่งต่ออัตโนมัติ กันวนไม่รู้จบ แต่ให้กดลองใหม่เอง
 */
export default function LoginPage() {
  return (
    <Suspense>
      <Redirect />
    </Suspense>
  );
}

function Redirect() {
  const params = useSearchParams();
  const callbackUrl = params.get("callbackUrl");
  const error = params.get("error");
  // safePath ใช้ window → เรียกเฉพาะใน effect/ตอนกด (ทำงานบน browser เท่านั้น)
  const go = () => signIn("keycloak", { callbackUrl: safePath(callbackUrl) });

  useEffect(() => {
    if (!error) void signIn("keycloak", { callbackUrl: safePath(callbackUrl) });
  }, [error, callbackUrl]);

  return (
    <main className="grid min-h-screen place-items-center px-4">
      <div className="flex w-full max-w-sm flex-col items-center gap-5 rounded-3xl border border-border bg-surface p-8 text-center">
        <span className="brand-mark grid size-12 place-items-center rounded-[14px]">
          <BrandMark size={36} />
        </span>
        {error ? (
          <>
            <div className="flex flex-col gap-2">
              <h1 className="text-xl font-semibold">เข้าสู่ระบบไม่สำเร็จ</h1>
              <p className="text-sm text-muted">ระบบยืนยันตัวตนไม่ตอบสนองหรือการเข้าสู่ระบบถูกยกเลิก ลองอีกครั้งได้เลย</p>
            </div>
            <Button variant="cta" onClick={go} className="w-full">ลองเข้าสู่ระบบอีกครั้ง</Button>
          </>
        ) : (
          <p role="status" className="flex items-center gap-2 text-muted">
            <Loader2 size={18} className="animate-spin motion-reduce:animate-none" aria-hidden />
            กำลังพาไปหน้าเข้าสู่ระบบ…
          </p>
        )}
      </div>
    </main>
  );
}

/** รับเฉพาะ path ภายในเว็บเราเอง (ขึ้นต้นด้วย / แต่ไม่ใช่ //) — กัน open redirect ไปเว็บอื่น */
function safePath(url: string | null) {
  if (!url) return "/";
  try {
    const u = new URL(url, window.location.origin);
    return u.origin === window.location.origin ? u.pathname + u.search : "/";
  } catch {
    return "/";
  }
}
