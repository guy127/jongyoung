import axios, { isAxiosError } from "axios";
import { getSession, signIn } from "next-auth/react";

import type { ApiError } from "./types";

export const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://api.jongyoung.localhost/api/v1";

/** axios สำหรับ client component — แนบ access token จาก session ของ next-auth ทุกคำขอ */
export const api = axios.create({ baseURL: API_BASE, timeout: 10_000 });

api.interceptors.request.use(async (config) => {
  const session = await getSession(); // next-auth ต่ออายุ token ให้ใน callback jwt ก่อนคืน session
  if (session?.accessToken) config.headers.Authorization = `Bearer ${session.accessToken}`;
  return config;
});

api.interceptors.response.use(
  (res) => res,
  (err) => {
    // 401 = ไม่มี/หมดอายุ → ให้ login ใหม่แล้วกลับมาหน้าเดิม
    if (isAxiosError(err) && err.response?.status === 401) void signIn("keycloak");
    return Promise.reject(err);
  },
);

/** ดึง { code, message, details } จาก error ของ API (ไม่ใช่ error ของ API → null) */
export function apiError(err: unknown): ApiError | null {
  if (isAxiosError(err) && err.response?.data?.error) return err.response.data.error as ApiError;
  return null;
}

/** GET จาก server component — ข้อมูลสาธารณะ ไม่ต้องใช้ token และไม่ cache (ที่นั่งเปลี่ยนตลอด) */
export async function serverGet<T>(path: string): Promise<T | null> {
  const res = await fetch(`${API_BASE}${path}`, { cache: "no-store" });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`API ${res.status} ${path}`);
  return res.json() as Promise<T>;
}
