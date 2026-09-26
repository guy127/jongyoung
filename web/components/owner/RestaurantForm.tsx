"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { ImagePlus, Loader2 } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { apiError } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import type { ApiError, Restaurant } from "@/lib/types";
import { useImages, useSaveRestaurant } from "@/services/restaurants";

const clocks = Array.from({ length: 48 }, (_, i) => `${String(Math.floor(i / 2)).padStart(2, "0")}:${i % 2 ? "30" : "00"}`);

// กติกาเดียวกับ API (ข้อ 5.1) — ตรวจที่หน้าเว็บเพื่อ UX เท่านั้น API ตรวจซ้ำเสมอ
const schema = z.object({
  name: z.string().trim().min(1, "ต้องมีชื่อร้าน").max(120),
  cuisine: z.string().trim().max(60),
  address: z.string().trim().min(1, "ต้องมีที่อยู่").max(300),
  description: z.string().max(2000),
  seats: z.number({ message: "ใส่จำนวนที่นั่ง" }).int().min(1, "อย่างน้อย 1 ที่").max(1000),
  open_time: z.string(),
  close_time: z.string(),
  cancel_before_minutes: z.number({ message: "ใส่จำนวนนาที" }).int().min(30, "ขั้นต่ำ 30 นาที").max(1440),
  image_urls: z.string(),
});
type Values = z.infer<typeof schema>;

const input = "h-9 rounded-md border border-border-strong bg-surface px-2.5 text-text";

export default function RestaurantForm({ restaurant, onDone }: { restaurant?: Restaurant; onDone: () => void }) {
  const save = useSaveRestaurant();
  const [serverError, setServerError] = useState<ApiError | null>(null);
  const { register, handleSubmit, formState: { errors } } = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: restaurant
      ? { ...restaurant, image_urls: "" }
      : { name: "", cuisine: "", address: "", description: "", seats: 10, open_time: "11:00", close_time: "22:00", cancel_before_minutes: 30, image_urls: "" },
  });

  const onSubmit = handleSubmit(async ({ image_urls, ...values }) => {
    const urls = image_urls.split("\n").map((s) => s.trim()).filter(Boolean);
    if (!restaurant && urls.length === 0) return setServerError({ code: "IMAGE_REQUIRED", message: "" });
    setServerError(null);
    try {
      await save.mutateAsync({ id: restaurant?.id, input: { ...values, image_urls: restaurant ? undefined : urls } });
      onDone();
    } catch (err) {
      setServerError(apiError(err) ?? { code: "NETWORK", message: "" });
    }
  });

  const field = (label: string, el: React.ReactNode, err?: string, wide = false) => (
    <label className={`flex flex-col gap-1 font-medium ${wide ? "sm:col-span-2" : ""}`}>
      {label}
      {el}
      {err && <span className="font-normal text-full">{err}</span>}
    </label>
  );

  return (
    <form onSubmit={onSubmit} className="flex flex-col border border-border bg-surface">
      <h2 className="border-b border-border px-4 py-3 text-base font-semibold">{restaurant ? `แก้ไข: ${restaurant.name}` : "เพิ่มร้านใหม่"}</h2>
      <div className="grid gap-4 p-4 sm:grid-cols-2">
        {field("ชื่อร้าน *", <input {...register("name")} className={input} />, errors.name?.message)}
        {field("ประเภทอาหาร", <input {...register("cuisine")} className={input} placeholder="เช่น อาหารไทย" />)}
        {field("ที่อยู่ *", <input {...register("address")} className={input} />, errors.address?.message, true)}
        {field("คำอธิบาย", <textarea {...register("description")} rows={2} className={`${input} h-auto py-2`} />, undefined, true)}
        {field("จำนวนที่นั่ง *", <input type="number" {...register("seats", { valueAsNumber: true })} className={input} />, errors.seats?.message)}
        {field("ยกเลิกได้ก่อนเวลาจอง (นาที) *", <input type="number" {...register("cancel_before_minutes", { valueAsNumber: true })} className={input} />, errors.cancel_before_minutes?.message)}
        {field("เปิด *", <select {...register("open_time")} className={input}>{clocks.map((c) => <option key={c}>{c}</option>)}</select>)}
        {field("ปิด *", <select {...register("close_time")} className={input}>{clocks.map((c) => <option key={c}>{c}</option>)}</select>)}
        <p className="text-muted sm:col-span-2">ปิดน้อยกว่าเปิด = ข้ามเที่ยงคืน (เช่น 18:00–02:00) · เปิดเท่ากับปิด = 24 ชม.</p>
        {!restaurant && field("URL รูป * (บรรทัดละ 1 รูป, รูปแรกเป็นรูปปก)", <textarea {...register("image_urls")} rows={3} className={`${input} h-auto py-2`} placeholder="https://..." />, undefined, true)}
      </div>
      {restaurant && <ImageManager restaurant={restaurant} />}
      {serverError && (
        <p role="alert" className="mx-4 mb-3 rounded-md border border-full-line bg-full-weak p-3 text-full">
          ✕ {errorMessage(serverError.code === "NETWORK" ? null : serverError)} {serverError.code !== "NETWORK" && `(${serverError.code})`}
        </p>
      )}
      <div className="flex justify-end gap-2 border-t border-border bg-chip/40 px-4 py-3">
        <button type="button" onClick={onDone} className="obtn">ยกเลิก</button>
        <button type="submit" disabled={save.isPending} aria-busy={save.isPending || undefined} className="obtn obtn-primary">
          {save.isPending && <Loader2 size={16} className="animate-spin motion-reduce:animate-none" aria-hidden />}
          {save.isPending ? "กำลังบันทึก…" : "บันทึก"}
        </button>
      </div>
    </form>
  );
}

/** จัดการรูปของร้านที่มีอยู่แล้ว — ห้ามลบรูปสุดท้าย (API ตอบ IMAGE_REQUIRED) */
function ImageManager({ restaurant }: { restaurant: Restaurant }) {
  const { add, remove } = useImages(restaurant.id);
  const [url, setUrl] = useState("");
  const err = apiError(add.error ?? remove.error);
  return (
    <fieldset className="mx-4 mb-4 flex flex-col gap-2">
      <legend className="pb-1 font-medium">รูป (URL) — อย่างน้อย 1 รูป แถวแรกคือรูปปก</legend>
      {restaurant.images.map((img) => (
        <div key={img.id} className="flex items-center gap-2">
          <span className="flex-1 truncate text-muted">{img.url}</span>
          <button type="button" disabled={restaurant.images.length <= 1 || remove.isPending} onClick={() => remove.mutate(img.id)}
            title={restaurant.images.length <= 1 ? "ลบรูปสุดท้ายไม่ได้" : undefined}
            className="obtn obtn-sm obtn-danger">ลบ</button>
        </div>
      ))}
      <div className="flex gap-2">
        <input value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://..." aria-label="URL รูปใหม่" className={`${input} flex-1`} />
        <button type="button" disabled={!url || add.isPending} onClick={() => add.mutate(url, { onSuccess: () => setUrl("") })}
          className="obtn"><ImagePlus size={16} aria-hidden />เพิ่มรูป</button>
      </div>
      {err && <p role="alert" className="text-full">✕ {errorMessage(err)}</p>}
    </fieldset>
  );
}
