"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SessionProvider, signOut, useSession } from "next-auth/react";
import { useEffect, useState } from "react";

// refresh token ใช้ไม่ได้แล้ว (ถูก revoke/หมดอายุ) → ออกจากระบบ ให้ผู้ใช้ login ใหม่
function SessionWatcher() {
  const { data } = useSession();
  useEffect(() => {
    if (data?.error === "RefreshAccessTokenError") void signOut({ callbackUrl: "/" });
  }, [data?.error]);
  return null;
}

export default function Providers({ children }: { children: React.ReactNode }) {
  // สร้าง QueryClient ครั้งเดียวต่อ browser (useState กันสร้างใหม่ทุก render)
  const [queryClient] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: true } } }),
  );
  return (
    <SessionProvider>
      <QueryClientProvider client={queryClient}>
        <SessionWatcher />
        {children}
      </QueryClientProvider>
    </SessionProvider>
  );
}
