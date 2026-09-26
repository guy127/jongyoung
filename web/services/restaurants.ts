import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { Availability, NextAvailable, Restaurant, RestaurantImage } from "@/lib/types";

export type RestaurantInput = {
  name: string;
  description: string;
  cuisine: string;
  address: string;
  map_url: string;
  seats: number;
  open_time: string;
  close_time: string;
  break_start: string; // "" = ไม่มีช่วงพัก
  break_end: string;
  closed_weekdays: number[];
  cancel_before_minutes: number;
  image_urls?: string[];
};

export const useRestaurant = (id: string | undefined) =>
  useQuery({
    queryKey: ["restaurant", id],
    queryFn: async () => (await api.get<Restaurant>(`/restaurants/${id}`)).data,
    enabled: !!id,
  });

// ที่นั่งเปลี่ยนได้ทุกวินาที → ไม่ใช้ cache เก่า (staleTime 0) และดึงใหม่เมื่อกลับมาที่แท็บ
export const useAvailability = (id: string, date: string) =>
  useQuery({
    queryKey: ["availability", id, date],
    queryFn: async () => (await api.get<Availability>(`/restaurants/${id}/availability`, { params: { date } })).data,
    staleTime: 0,
  });

export const useNextAvailable = (id: string, date: string, time: string, party: number, enabled: boolean) =>
  useQuery({
    queryKey: ["next-available", id, date, time, party],
    queryFn: async () =>
      (await api.get<NextAvailable>(`/restaurants/${id}/next-available`, { params: { date, time, party_size: party } })).data,
    enabled,
  });

export function useSaveRestaurant() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id?: string; input: RestaurantInput }) =>
      id
        ? (await api.put<Restaurant>(`/restaurants/${id}`, input)).data
        : (await api.post<Restaurant>(`/restaurants`, input)).data,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["restaurant"] });
      void qc.invalidateQueries({ queryKey: ["me"] });
    },
  });
}

export function useDeleteRestaurant() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/restaurants/${id}`),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["me"] }),
  });
}

export function useImages(restaurantId: string) {
  const qc = useQueryClient();
  const refresh = () => qc.invalidateQueries({ queryKey: ["restaurant", restaurantId] });
  const add = useMutation({
    mutationFn: async (url: string) => (await api.post<RestaurantImage>(`/restaurants/${restaurantId}/images`, { url })).data,
    onSuccess: refresh,
  });
  const remove = useMutation({
    mutationFn: async (imageId: string) => api.delete(`/restaurants/${restaurantId}/images/${imageId}`),
    onSuccess: refresh,
  });
  return { add, remove };
}
