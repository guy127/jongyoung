import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { Board, Booking } from "@/lib/types";

/** ส่งวันทำการ + เวลาบนนาฬิกา — server แปลงเป็นเวลาจริงเอง (รองรับร้านข้ามคืน) */
export type BookingInput = {
  restaurant_id?: string;
  date: string;
  start_time: string;
  end_time: string;
  party_size: number;
};

export const useBooking = (id: string) =>
  useQuery({ queryKey: ["booking", id], queryFn: async () => (await api.get<Booking>(`/bookings/${id}`)).data });

export const useMyBookings = (status: "upcoming" | "past" | "cancelled") =>
  useQuery({
    queryKey: ["my-bookings", status],
    queryFn: async () => (await api.get<Booking[]>(`/me/bookings`, { params: { status } })).data,
  });

export const useBoard = (restaurantId: string | undefined, date: string) =>
  useQuery({
    queryKey: ["board", restaurantId, date],
    queryFn: async () => (await api.get<Board>(`/restaurants/${restaurantId}/bookings`, { params: { date } })).data,
    enabled: !!restaurantId,
  });

// การจอง: ห้าม optimistic update — รอผลจริงจาก server เสมอ (CLAUDE.md ข้อ 7)
export function useSaveBooking() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id?: string; input: BookingInput }) =>
      id ? (await api.put<Booking>(`/bookings/${id}`, input)).data : (await api.post<Booking>(`/bookings`, input)).data,
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: ["availability"] });
      void qc.invalidateQueries({ queryKey: ["my-bookings"] });
    },
  });
}

export function useCancelBooking() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/bookings/${id}`),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["my-bookings"] });
      void qc.invalidateQueries({ queryKey: ["booking"] });
    },
  });
}
