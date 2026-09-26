import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { Closure } from "@/lib/types";

/** บางช่วงของวันทำการ (date + start_time + end_time) หรือทั้งวัน/หลายวัน (from_date + to_date) */
export type ClosureInput = {
  date?: string;
  start_time?: string;
  end_time?: string;
  from_date?: string;
  to_date?: string;
  reason: string;
  confirm_booking_ids?: string[]; // ส่งเมื่อยืนยันรายการจองที่จะถูกยกเลิก (ได้จาก 409 CLOSURE_AFFECTS_BOOKINGS)
};

/** ช่วงปิดที่ยังไม่จบ (สาธารณะ) */
export const useClosures = (restaurantId: string | undefined) =>
  useQuery({
    queryKey: ["closures", restaurantId],
    queryFn: async () => (await api.get<Closure[]>(`/restaurants/${restaurantId}/closures`)).data,
    enabled: !!restaurantId,
  });

// ปิด/เปิดร้านกระทบเวลาว่าง บอร์ด และการจอง → โหลดใหม่หมด
const refreshAfterChange = (qc: ReturnType<typeof useQueryClient>) =>
  Promise.all(["closures", "availability", "next-available", "board"].map((key) => qc.invalidateQueries({ queryKey: [key] })));

export function useCreateClosure(restaurantId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: ClosureInput) => (await api.post<Closure>(`/restaurants/${restaurantId}/closures`, input)).data,
    onSuccess: () => refreshAfterChange(qc),
  });
}

export function useDeleteClosure(restaurantId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (closureId: string) => {
      await api.delete(`/restaurants/${restaurantId}/closures/${closureId}`);
    },
    onSuccess: () => refreshAfterChange(qc),
  });
}
