import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // standalone = build ออกมาเป็นโฟลเดอร์ที่รันด้วย node server.js ได้เลย → Docker image เล็ก
  output: "standalone",
  images: {
    // เจ้าของร้านวาง URL รูปจากเว็บไหนก็ได้ (ไม่มีระบบอัปโหลด) จึงอนุญาตทุก host ที่เป็น https
    remotePatterns: [{ protocol: "https", hostname: "**" }],
  },
};

export default nextConfig;
