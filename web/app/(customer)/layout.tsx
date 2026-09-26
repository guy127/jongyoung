import Navbar from "@/containers/Navbar";

export default function CustomerLayout({ children }: { children: React.ReactNode }) {
  return (
    <>
      <Navbar />
      <main className="mx-auto w-[min(1120px,calc(100%-2rem))] pb-16 pt-6">{children}</main>
      <footer className="mx-auto w-[min(1120px,calc(100%-2rem))] border-t border-border py-8 text-sm text-soft">
        © 2026 จองยัง — PEA DevPool Final Project · Next.js · Go · PostgreSQL · Keycloak
      </footer>
    </>
  );
}
