"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { ImagePlus, Loader2 } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { apiError } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { isGoogleMapsUrl, WEEKDAY_ORDER, WEEKDAYS } from "@/lib/format";
import type { ApiError, Restaurant } from "@/lib/types";
import { useImages, useSaveRestaurant } from "@/services/restaurants";

const clocks = Array.from({ length: 48 }, (_, i) => `${String(Math.floor(i / 2)).padStart(2, "0")}:${i % 2 ? "30" : "00"}`);

// กติกาเดียวกับ API (ข้อ 5.1) — ตรวจที่หน้าเว็บเพื่อ UX เท่านั้น API ตรวจซ้ำเสมอ
const schema = z.object({
  name: z.string().trim().min(1, "ต้องมีชื่อร้าน").max(120),
  cuisine: z.string().trim().max(60),
  address: z.string().trim().min(1, "ต้องมีที่อยู่").max(300),
  map_url: z.string().trim().max(500).refine((s) => s === "" || isGoogleMapsUrl(s), "ต้องเป็นลิงก์ Google Maps (https)"),
  description: z.string().max(2000),
  seats: z.number({ message: "ใส่จำนวนที่นั่ง" }).int().min(1, "อย่างน้อย 1 ที่").max(1000),
  open_time: z.string(),
  close_time: z.string(),
  break_start: z.string(), // "" = ไม่มีช่วงพัก
  break_end: z.string(),
  // checkbox หลายตัวชื่อเดียวกัน react-hook-form คืนเป็น string[] ("0"–"6") → แปลงเป็นตัวเลขตอนส่ง
  closed_weekdays: z.array(z.string()).max(6, "ร้านต้องเปิดอย่างน้อย 1 วันต่อสัปดาห์"),
  cancel_before_minutes: z.number({ message: "ใส่จำนวนนาที" }).int().min(30, "ขั้นต่ำ 30 นาที").max(1440),
  image_urls: z.string(),
}).refine((v) => (v.break_start === "") === (v.break_end === ""), { path: ["break_end"], message: "กรอกช่วงพักให้ครบทั้งเวลาเริ่มและจบ หรือเว้นว่างทั้งคู่" });
type Values = z.infer<typeof schema>;

const input = "h-9 rounded-md border border-border-strong bg-surface px-2.5 text-text";

export default function RestaurantForm({ restaurant, onDone }: { restaurant?: Restaurant; onDone: () => void }) {
  const save = useSaveRestaurant();
  const [serverError, setServerError] = useState<ApiError | null>(null);
  const { register, handleSubmit, formState: { errors } } = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: restaurant
      ? { ...restaurant, closed_weekdays: restaurant.closed_weekdays.map(String), image_urls: "" }
      : { name: "", cuisine: "", address: "", map_url: "", description: "", seats: 10, open_time: "11:00", close_time: "22:00", break_start: "", break_end: "", closed_weekdays: [], cancel_before_minutes: 30, image_urls: "" },
  });

  const onSubmit = handleSubmit(async ({ image_urls, closed_weekdays, ...values }) => {
    const urls = image_urls.split("\n").map((s) => s.trim()).filter(Boolean);
    if (!restaurant && urls.length === 0) return setServerError({ code: "IMAGE_REQUIRED", message: "" });
    setServerError(null);
    try {
      await save.mutateAsync({ id: restaurant?.id, input: { ...values, closed_weekdays: closed_weekdays.map(Number), image_urls: restaurant ? undefined : urls } });
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
    // min-w-0: ไม่งั้น grid item กว้างตาม URL รูปที่ยาวที่สุด → จอแคบแล้วปุ่มบันทึกหลุดขอบขวา
    <form onSubmit={onSubmit} className="flex min-w-0 flex-col border border-border bg-surface">
      <h2 className="border-b border-border px-4 py-3 text-base font-semibold">{restaurant ? `แก้ไข: ${restaurant.name}` : "เพิ่มร้านใหม่"}</h2>
      <div className="grid gap-4 p-4 sm:grid-cols-2">
        {field("ชื่อร้าน *", <input {...register("name")} className={input} />, errors.name?.message)}
        {field("ประเภทอาหาร", <input {...register("cuisine")} className={input} placeholder="เช่น อาหารไทย" />)}
        {field("ที่อยู่ *", <input {...register("address")} className={input} />, errors.address?.message, true)}
        {field("ลิงก์ Google Maps (ไม่บังคับ — ไม่ใส่จะค้นแผนที่จากที่อยู่)",
          <input {...register("map_url")} inputMode="url" className={input} placeholder="https://maps.app.goo.gl/..." />, errors.map_url?.message, true)}
        {field("คำอธิบาย", <textarea {...register("description")} rows={2} className={`${input} h-auto py-2`} />, undefined, true)}
        {field("จำนวนที่นั่ง *", <input type="number" {...register("seats", { valueAsNumber: true })} className={input} />, errors.seats?.message)}
        {field("ยกเลิกได้ก่อนเวลาจอง (นาที) *", <input type="number" {...register("cancel_before_minutes", { valueAsNumber: true })} className={input} />, errors.cancel_before_minutes?.message)}
        {field("เปิด *", <select {...register("open_time")} className={input}>{clocks.map((c) => <option key={c}>{c}</option>)}</select>)}
        {field("ปิด *", <select {...register("close_time")} className={input}>{clocks.map((c) => <option key={c}>{c}</option>)}</select>)}
        {field("พักตั้งแต่ (ไม่บังคับ)", <select {...register("break_start")} className={input}><option value="">ไม่มีช่วงพัก</option>{clocks.map((c) => <option key={c}>{c}</option>)}</select>)}
        {field("ถึง", <select {...register("break_end")} className={input}><option value="">ไม่มีช่วงพัก</option>{clocks.map((c) => <option key={c}>{c}</option>)}</select>, errors.break_end?.message)}
        <p className="text-muted sm:col-span-2">ปิดน้อยกว่าเปิด = ข้ามเที่ยงคืน (เช่น 18:00–02:00) · เปิดเท่ากับปิด = 24 ชม. · ช่วงพัก = รับจองไม่ได้ช่วงนั้น เช่น เปิด 11:00–22:00 พัก 14:00–17:00</p>
        <fieldset className="flex flex-col gap-2 sm:col-span-2">
          <legend className="pb-1 font-medium">วันปิดประจำสัปดาห์</legend>
          <div className="flex flex-wrap gap-2">
            {WEEKDAY_ORDER.map((d) => (
              <label key={d} className="inline-flex min-h-9 items-center gap-1.5 rounded-md border border-border-strong px-2.5">
                <input type="checkbox" value={String(d)} {...register("closed_weekdays")} />
                {WEEKDAYS[d]}
              </label>
            ))}
          </div>
          <p className="text-muted">นับตามรอบที่เริ่มเปิดในวันนั้น — ร้าน 18:00–02:00 ปิดวันจันทร์ ตีหนึ่งของเช้าวันจันทร์ยังเป็นรอบวันอาทิตย์</p>
          {errors.closed_weekdays && <span className="text-full">{errors.closed_weekdays.message}</span>}
        </fieldset>
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
    // fieldset ของ browser ตั้ง min-width: min-content ไว้ → ต้องใส่ min-w-0 ให้ URL ยาว ๆ ถูกตัดด้วย … ได้
    <fieldset className="mx-4 mb-4 flex min-w-0 flex-col gap-2">
      <legend className="pb-1 font-medium">รูป (URL) — อย่างน้อย 1 รูป แถวแรกคือรูปปก</legend>
      {restaurant.images.map((img) => (
        <div key={img.id} className="flex items-center gap-2">
          <span className="min-w-0 flex-1 truncate text-muted" title={img.url}>{img.url}</span>
          <button type="button" disabled={restaurant.images.length <= 1 || remove.isPending} onClick={() => remove.mutate(img.id)}
            title={restaurant.images.length <= 1 ? "ลบรูปสุดท้ายไม่ได้" : undefined}
            className="obtn obtn-sm obtn-danger">ลบ</button>
        </div>
      ))}
      <div className="flex gap-2">
        <input value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://..." aria-label="URL รูปใหม่" className={`${input} min-w-0 flex-1`} />
        <button type="button" disabled={!url || add.isPending} onClick={() => add.mutate(url, { onSuccess: () => setUrl("") })}
          className="obtn"><ImagePlus size={16} aria-hidden />เพิ่มรูป</button>
      </div>
      {err && <p role="alert" className="text-full">✕ {errorMessage(err)}</p>}
    </fieldset>
  );
}
