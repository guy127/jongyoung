import OwnerNavbar from "@/containers/OwnerNavbar";

// /owner/* เป็นเครื่องมือทำงาน: ตัวอักษร 14px พื้นทึบ ไม่มี gradient/glass (CLAUDE.md ข้อ 8.5)
export default function OwnerLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-bg text-sm">
      <OwnerNavbar />
      <main className="mx-auto max-w-[1216px] px-4 py-6">{children}</main>
    </div>
  );
}
