import type { Metadata } from "next";
import { Kanit, Prompt } from "next/font/google";

import Providers from "./providers";
import "./globals.css";

// Prompt ทั้งระบบ 3 น้ำหนัก + Kanit น้ำหนักเดียวสำหรับโลโก้/หัวข้อ hero (CLAUDE.md ข้อ 8.1)
const prompt = Prompt({ subsets: ["thai", "latin"], weight: ["400", "500", "600"], variable: "--font-prompt", display: "swap" });
const kanit = Kanit({ subsets: ["thai", "latin"], weight: ["600"], variable: "--font-kanit", display: "swap" });

export const metadata: Metadata = {
  title: "จองยัง — จองโต๊ะร้านอาหาร",
  description: "ดูเวลาว่างจริงของร้านอาหาร แล้วจองได้ใน 2 แท็ป",
};

// อ่านธีมที่เคยเลือกก่อน render ครั้งแรก กันหน้าจอกระพริบจากสว่างเป็นมืด (สคริปต์คงที่ ไม่มีข้อมูลผู้ใช้)
const themeScript = `try{if(localStorage.getItem("jy-theme")==="night")document.documentElement.dataset.theme="night"}catch(e){}`;

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="th" className={`${prompt.variable} ${kanit.variable}`} suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className="min-h-screen font-sans antialiased">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
