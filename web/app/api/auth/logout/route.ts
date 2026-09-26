import { getServerSession } from "next-auth";
import { NextResponse, type NextRequest } from "next/server";

import { authOptions } from "@/lib/auth/auth-options";

// ออกจากระบบทั้งสองฝั่ง: ลบ cookie ของ next-auth แล้วส่งไป end_session ของ Keycloak
// (ถ้าลบแค่ cookie ผู้ใช้กด "เข้าสู่ระบบ" อีกครั้งจะเข้าได้เลยเพราะ session ที่ Keycloak ยังอยู่)
export async function GET(request: NextRequest) {
  const session = await getServerSession(authOptions); // ต้องอ่านก่อนลบ cookie
  const home = process.env.NEXTAUTH_URL ?? "http://jongyoung.localhost";

  let destination = home;
  if (session?.idToken) {
    const url = new URL(`${process.env.KEYCLOAK_ISSUER}/protocol/openid-connect/logout`);
    url.searchParams.set("id_token_hint", session.idToken);
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
