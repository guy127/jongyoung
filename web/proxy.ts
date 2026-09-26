import { withAuth } from "next-auth/middleware";

// หน้าที่ต้อง login — ยังไม่ login จะถูกพาไปหน้าเข้าสู่ระบบ แล้วกลับมาหน้าเดิม (callbackUrl)
// (เป็นเรื่อง UX เท่านั้น API ตรวจ token เองทุกคำขอ)
// Next 16 ต้องการ function ที่ export ตรง ๆ จากไฟล์นี้ จึงเรียก withAuth() แทนการ re-export
export default withAuth({});

export const config = {
  matcher: ["/me/:path*", "/owner/:path*", "/bookings/:path*"],
};
