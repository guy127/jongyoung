import BrandMark from "@/components/bases/BrandMark";
import { LinkButton } from "@/components/bases/ui";

/** URL ที่ไม่ตรงกับหน้าไหนเลย — แทนหน้า 404 ภาษาอังกฤษของ Next (อยู่ใต้ root layout จึงไม่มี navbar) */
export default function NotFound() {
  return (
    <main className="grid min-h-screen place-items-center px-4">
      <div className="flex w-full max-w-sm flex-col items-center gap-5 rounded-3xl border border-border bg-surface p-8 text-center">
        <span className="brand-mark grid size-12 place-items-center rounded-[14px]">
          <BrandMark size={36} />
        </span>
        <div className="flex flex-col gap-2">
          <h1 className="text-xl font-semibold">ไม่พบหน้านี้</h1>
          <p className="text-sm text-muted">ลิงก์อาจพิมพ์ผิดหรือหน้านี้ถูกย้ายไปแล้ว</p>
        </div>
        <LinkButton href="/" variant="cta" className="w-full">กลับไปค้นหาร้าน</LinkButton>
      </div>
    </main>
  );
}
