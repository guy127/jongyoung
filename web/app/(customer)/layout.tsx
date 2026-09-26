import Navbar from "@/containers/Navbar";

// ความกว้างหน้า 1120px ขอบข้าง 16px (มือถือ) / 24px (desktop) — ข้อ 8.3
const frame = "mx-auto w-full max-w-[1168px] px-4 sm:px-6";

export default function CustomerLayout({ children }: { children: React.ReactNode }) {
  return (
    <>
      <Navbar />
      <main className={`${frame} pb-20 pt-8 sm:pt-10`}>{children}</main>
      <footer className={frame}>
        <p className="border-t border-border py-8 text-sm text-soft">
          © 2026 จองยัง — PEA DevPool Final Project · Next.js · Go · PostgreSQL · Keycloak
        </p>
      </footer>
    </>
  );
}
