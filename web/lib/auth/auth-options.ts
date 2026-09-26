import type { NextAuthOptions } from "next-auth";
import type { JWT } from "next-auth/jwt";
import KeycloakProvider from "next-auth/providers/keycloak";

const issuer = process.env.KEYCLOAK_ISSUER!;

/** ขอ access token ใหม่ด้วย refresh token (ถ้าไม่ผ่าน ใส่ error ไว้ให้หน้าเว็บสั่ง signOut) */
async function refreshAccessToken(token: JWT): Promise<JWT> {
  try {
    const res = await fetch(`${issuer}/protocol/openid-connect/token`, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        grant_type: "refresh_token",
        client_id: process.env.KEYCLOAK_CLIENT_ID!,
        client_secret: process.env.KEYCLOAK_CLIENT_SECRET!,
        refresh_token: token.refreshToken!,
      }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error ?? "refresh failed");
    return {
      ...token,
      accessToken: data.access_token,
      idToken: data.id_token ?? token.idToken,
      refreshToken: data.refresh_token ?? token.refreshToken,
      expiresAt: Math.floor(Date.now() / 1000) + data.expires_in,
      error: undefined,
    };
  } catch {
    return { ...token, error: "RefreshAccessTokenError" };
  }
}

export const authOptions: NextAuthOptions = {
  providers: [
    KeycloakProvider({
      clientId: process.env.KEYCLOAK_CLIENT_ID!,
      clientSecret: process.env.KEYCLOAK_CLIENT_SECRET!,
      issuer,
    }),
  ],
  callbacks: {
    // jwt ถูกเรียกทุกครั้งที่อ่าน session — ที่นี่คือจุดเดียวที่ต่ออายุ token
    async jwt({ token, account }) {
      if (account) {
        // login ครั้งแรก: เก็บ token จาก Keycloak ไว้ใน cookie ของ next-auth (เข้ารหัสด้วย NEXTAUTH_SECRET)
        return {
          ...token,
          accessToken: account.access_token,
          refreshToken: account.refresh_token,
          idToken: account.id_token,
          expiresAt: account.expires_at,
        };
      }
      // ยังเหลือเกิน 30 วินาที → ใช้ของเดิม
      if (token.expiresAt && Date.now() / 1000 < token.expiresAt - 30) return token;
      return refreshAccessToken(token);
    },
    async session({ session, token }) {
      session.accessToken = token.accessToken;
      session.idToken = token.idToken;
      session.error = token.error;
      return session;
    },
  },
};
