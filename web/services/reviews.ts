import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { isAxiosError } from "axios";

import { api } from "@/lib/api";
import type { Page, Review } from "@/lib/types";

export type ReviewInput = { score: number; body: string };

export const useReviews = (restaurantId: string, page: number, initial?: Page<Review>) =>
  useQuery({
    queryKey: ["reviews", restaurantId, page],
    queryFn: async () => (await api.get<Page<Review>>(`/restaurants/${restaurantId}/reviews`, { params: { page, limit: 5 } })).data,
    initialData: page === 1 ? initial : undefined,
  });

/** รีวิวของฉันในร้านนี้ — ยังไม่มีคืน null (API ตอบ 404) เพื่อให้รู้ว่าต้อง POST หรือ PUT */
export const useMyReview = (restaurantId: string, enabled: boolean) =>
  useQuery({
    queryKey: ["my-review", restaurantId],
    enabled,
    queryFn: async () => {
      try {
        return (await api.get<Review>(`/restaurants/${restaurantId}/reviews/mine`)).data;
      } catch (err) {
        if (isAxiosError(err) && err.response?.status === 404) return null;
        throw err;
      }
    },
  });

export function useSaveReview(restaurantId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ exists, input }: { exists: boolean; input: ReviewInput }) =>
      exists
        ? (await api.put<Review>(`/restaurants/${restaurantId}/reviews`, input)).data
        : (await api.post<Review>(`/restaurants/${restaurantId}/reviews`, input)).data,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["reviews", restaurantId] });
      void qc.invalidateQueries({ queryKey: ["my-review", restaurantId] });
    },
  });
}

export function useDeleteReview(restaurantId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => api.delete(`/restaurants/${restaurantId}/reviews`),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["reviews", restaurantId] });
      void qc.invalidateQueries({ queryKey: ["my-review", restaurantId] });
    },
  });
}
