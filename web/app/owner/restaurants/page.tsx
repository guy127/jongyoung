"use client";

import { useQueries } from "@tanstack/react-query";
import Link from "next/link";
import { useState } from "react";

import RestaurantForm from "@/components/owner/RestaurantForm";
import { api, apiError } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { ratingLabel } from "@/lib/format";
import type { Restaurant } from "@/lib/types";
import { useMe } from "@/services/me";
import { useDeleteRestaurant } from "@/services/restaurants";

export default function OwnerRestaurantsPage() {
  const me = useMe();
  const ids = me.data?.restaurants.map((r) => r.id) ?? [];
  // เจ้าของร้านมีร้านไม่กี่ร้าน → ดึงรายละเอียดทีละร้านพร้อมกัน (ขนาน ไม่ใช่ต่อคิว)
  const details = useQueries({
    queries: ids.map((id) => ({ queryKey: ["restaurant", id], queryFn: async () => (await api.get<Restaurant>(`/restaurants/${id}`)).data })),
  });
  const restaurants = details.map((q) => q.data).filter((r): r is Restaurant => !!r);
  const [editing, setEditing] = useState<string | "new" | null>(null);
  const current = restaurants.find((r) => r.id === editing);
  const th = "px-3 py-2 text-left font-medium text-muted";
  const td = "px-3 py-2.5";

  return (
    <div className="flex flex-col gap-5">
      <div className="flex items-center justify-between">
        <h1 className="text-[22px] font-semibold">ร้านของฉัน ({ids.length})</h1>
        <button type="button" onClick={() => setEditing("new")} className="h-9 rounded-md border border-border-strong bg-surface px-3.5">+ เพิ่มร้าน</button>
      </div>

      {me.isLoading && <p className="text-muted">กำลังโหลด…</p>}
      {me.isError && <p role="alert" className="text-full">โหลดข้อมูลไม่สำเร็จ <button className="underline" onClick={() => me.refetch()}>ลองอีกครั้ง</button></p>}
      {me.data && ids.length === 0 && editing === null && (
        <p className="rounded-md border border-dashed border-border-strong bg-surface p-6 text-center">ยังไม่มีร้าน — กด “+ เพิ่มร้าน” เพื่อเริ่มรับจอง</p>
      )}

      {restaurants.length > 0 && (
        <div className="overflow-x-auto border border-border bg-surface">
          <table className="w-full border-collapse tabular">
            <thead className="bg-chip/50">
              <tr><th className={th}>ร้าน</th><th className={th}>ประเภท</th><th className={th}>ที่นั่ง</th><th className={th}>เวลาเปิด–ปิด</th><th className={th}>ยกเลิกก่อน</th><th className={th}>คะแนน</th><th className={th}><span className="sr-only">จัดการ</span></th></tr>
            </thead>
            <tbody>
              {restaurants.map((r) => (
                <tr key={r.id} className={`border-t border-border ${editing === r.id ? "bg-sel-weak" : ""}`}>
                  <td className={`${td} font-semibold`}>{r.name}</td>
                  <td className={td}>{r.cuisine || "—"}</td>
                  <td className={td}>{r.seats}</td>
                  <td className={td}>{r.open_24h ? "24 ชม." : `${r.open_time}–${r.close_time}`}{r.overnight && <span className="text-muted"> (ข้ามคืน)</span>}</td>
                  <td className={td}>{r.cancel_before_minutes} นาที</td>
                  <td className={td}>{ratingLabel(r.rating).text}</td>
                  <td className={`${td} text-right`}>
                    <Link href={`/owner/bookings?restaurant=${r.id}`} className="mr-3 text-sel underline">การจอง</Link>
                    <button type="button" onClick={() => setEditing(r.id)} className="h-8 rounded-md border border-border-strong px-2.5">แก้ไข</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {editing !== null && (
        <div className="grid gap-5 lg:grid-cols-[1fr_340px] lg:items-start">
          <RestaurantForm key={editing} restaurant={current} onDone={() => setEditing(null)} />
          <aside className="flex flex-col gap-3">
            <section className="border border-border bg-surface p-3.5">
              <h3 className="font-semibold">ระบบตรวจตอนบันทึก</h3>
              <ol className="list-decimal pl-5 leading-7 text-muted">
                <li>ที่นั่งต้องไม่น้อยกว่าคนในร้านสูงสุดของการจองที่จะถึง</li>
                <li>เวลาเปิด–ปิดใหม่ต้องครอบการจองที่จะถึงทั้งหมด</li>
                <li>ยกเลิกก่อนเวลาจองอย่างน้อย 30 นาที</li>
                <li>มีรูปอย่างน้อย 1 รูป</li>
              </ol>
            </section>
            {current && <DeleteRestaurant restaurant={current} onDeleted={() => setEditing(null)} />}
          </aside>
        </div>
      )}
    </div>
  );
}

function DeleteRestaurant({ restaurant, onDeleted }: { restaurant: Restaurant; onDeleted: () => void }) {
  const del = useDeleteRestaurant();
  const [confirm, setConfirm] = useState(false);
  return (
    <section className="flex flex-col gap-2 border border-full-line bg-surface p-3.5">
      <h3 className="font-semibold text-full">ลบร้าน</h3>
      <p className="text-muted">ร้านจะหายจากหน้าค้นหาทันที การจองที่ยังไม่ถึงเวลาจะถูกยกเลิกอัตโนมัติ ประวัติและรีวิวยังเก็บไว้</p>
      {!confirm ? (
        <button type="button" onClick={() => setConfirm(true)} className="h-9 self-start rounded-md border border-full px-3 text-full">ลบร้านนี้</button>
      ) : (
        <div className="flex gap-2">
          <button type="button" onClick={() => setConfirm(false)} className="h-9 rounded-md border border-border-strong px-3">ไม่ลบ</button>
          <button type="button" disabled={del.isPending} onClick={() => del.mutate(restaurant.id, { onSuccess: onDeleted })}
            className="h-9 rounded-md bg-full px-3 font-semibold text-white">ยืนยันลบ “{restaurant.name}”</button>
        </div>
      )}
      {del.isError && <p role="alert" className="text-full">✕ {errorMessage(apiError(del.error))}</p>}
    </section>
  );
}
