import { useQuery } from "@tanstack/react-query";
import { useSession } from "next-auth/react";

import { api } from "@/lib/api";
import type { Me } from "@/lib/types";

/** โปรไฟล์ + ร้านที่เป็นเจ้าของ — เรียกเฉพาะเมื่อ login แล้ว */
export function useMe() {
  const { status } = useSession();
  return useQuery({
    queryKey: ["me"],
    queryFn: async () => (await api.get<Me>("/me")).data,
    enabled: status === "authenticated",
    staleTime: 60_000,
  });
}
