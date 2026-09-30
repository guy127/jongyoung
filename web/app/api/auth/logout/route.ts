import { getToken } from "next-auth/jwt";
import { NextResponse, type NextRequest } from "next/server";

// ออกจากระบบทั้งสองฝั่ง: ลบ cookie ของ next-auth แล้วส่งไป end_session ของ Keycloak
// (ถ้าลบแค่ cookie ผู้ใช้กด "เข้าสู่ระบบ" อีกครั้งจะเข้าได้เลยเพราะ session ที่ Keycloak ยังอยู่)
export async function GET(request: NextRequest) {
  // อ่าน id token จาก cookie ของ next-auth ฝั่ง server ตรง ๆ (ต้องอ่านก่อนลบ cookie)
  // id token ไม่อยู่ใน session ที่ browser เห็น — ใช้แค่ที่นี่ จึงไม่ต้องส่งออกไปให้ JavaScript
  // Keycloak รับ id_token_hint ที่หมดอายุแล้วได้ (ตรวจแค่ลายเซ็น) จึงไม่ต้องต่ออายุก่อน
  const token = await getToken({ req: request });
  const home = process.env.NEXTAUTH_URL ?? "http://jongyoung.localhost";

  let destination = home;
  if (token?.idToken) {
    const url = new URL(`${process.env.KEYCLOAK_ISSUER}/protocol/openid-connect/logout`);
    url.searchParams.set("id_token_hint", token.idToken);
    url.searchParams.set("post_logout_redirect_uri", home);
    destination = url.toString();
  }

  const response = NextResponse.redirect(destination);
  // ลบทุก cookie ของ next-auth ที่ browser ส่งมา — session มี token 3 ตัวจนเกิน 4KB
  // next-auth จึงแบ่งเป็น next-auth.session-token.0, .1, ... ลบแค่ชื่อเดียวไม่พอ
  for (const { name } of request.cookies.getAll()) {
    if (name.startsWith("next-auth.")) response.cookies.delete(name);
  }
  return response;
}
