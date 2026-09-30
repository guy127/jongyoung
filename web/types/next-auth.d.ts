import "next-auth";
import "next-auth/jwt";

declare module "next-auth" {
  interface Session {
    accessToken?: string; // browser ใช้แนบ Bearer — id token/refresh token อยู่ใน JWT ฝั่ง server เท่านั้น
    error?: "RefreshAccessTokenError";
  }
}

declare module "next-auth/jwt" {
  interface JWT {
    accessToken?: string;
    refreshToken?: string;
    idToken?: string;
    expiresAt?: number;
    error?: "RefreshAccessTokenError";
  }
}
