import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useSession } from "next-auth/react";

import { api } from "@/lib/api";
import type { NotificationList } from "@/lib/types";

/** แจ้งเตือนของฉัน — poll ทุก 60 วินาที + ตอนกลับมาที่แท็บ (ง่ายกว่า WebSocket และช้ากว่ากันไม่กี่วินาที) */
export function useNotifications() {
  const { status } = useSession();
  return useQuery({
    queryKey: ["notifications"],
    queryFn: async () => (await api.get<NotificationList>("/me/notifications")).data,
    enabled: status === "authenticated",
    refetchInterval: 60_000,
    refetchOnWindowFocus: true,
  });
}

export function useMarkRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.put(`/me/notifications/${id}/read`);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notifications"] }),
  });
}

export function useMarkAllRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.put("/me/notifications/read-all");
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notifications"] }),
  });
}
