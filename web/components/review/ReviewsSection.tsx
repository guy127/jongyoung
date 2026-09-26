"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { House, Star } from "lucide-react";
import Link from "next/link";
import { signIn, useSession } from "next-auth/react";
import { useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";

import Rating from "@/components/bases/Rating";
import { Alert, Button, EmptyState } from "@/components/bases/ui";
import { apiError } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { fmtShortDate } from "@/lib/format";
import type { Page, Restaurant, Review } from "@/lib/types";
import { useMe } from "@/services/me";
import { useDeleteReview, useMyReview, useReviews, useSaveReview } from "@/services/reviews";

// schema เดียวกับที่ API ตรวจ (คะแนนจำนวนเต็ม 1–5, ข้อความไม่เกิน 2000)
const schema = z.object({
  score: z.number().int().min(1, "เลือกคะแนน 1–5 ดาว").max(5),
  body: z.string().max(2000, "ยาวได้ไม่เกิน 2000 ตัวอักษร"),
});
type FormValues = z.infer<typeof schema>;

export default function ReviewsSection({ restaurant, initial }: { restaurant: Restaurant; initial?: Page<Review> }) {
  const { status } = useSession();
  const { data: me } = useMe();
  const [page, setPage] = useState(1);
  const list = useReviews(restaurant.id, page, initial);
  const isOwner = me?.id === restaurant.owner_id;
  const mine = useMyReview(restaurant.id, status === "authenticated" && !!me && !isOwner);
  const totalPages = list.data ? Math.ceil(list.data.total / list.data.limit) : 1;

  return (
    <section aria-labelledby="rev-h" className="flex flex-col gap-4">
      <h2 id="rev-h" className="flex flex-wrap items-center gap-2 text-xl font-semibold">
        รีวิว <span className="text-base font-normal text-muted"><Rating rating={restaurant.rating} /></span>
      </h2>

      {status === "unauthenticated" && (
        <Button className="self-start" onClick={() => signIn("keycloak")}>เข้าสู่ระบบเพื่อรีวิว</Button>
      )}
      {/* ร้านของตัวเอง: ไม่ให้กดแล้วเด้ง 403 แต่บอกเหตุผลตั้งแต่แรก (API ยังกัน 403 อยู่เสมอ) */}
      {isOwner && (
        <div className="flex gap-3 rounded-2xl bg-chip p-4">
          <House size={20} className="mt-0.5 shrink-0" aria-hidden />
          <div>
            <p className="font-semibold">นี่คือร้านของคุณ</p>
            <p className="text-sm text-muted">เจ้าของร้านรีวิวร้านตัวเองไม่ได้ เพื่อให้คะแนนเป็นกลางสำหรับลูกค้า</p>
            <Link href="/owner/restaurants" className="text-sm text-sel underline">ไปจัดการร้านนี้</Link>
          </div>
        </div>
      )}
      {mine.isSuccess && <ReviewForm restaurantId={restaurant.id} existing={mine.data} />}

      {list.isError && <Alert tone="full" title="โหลดรีวิวไม่สำเร็จ" />}
      {list.data?.items.length === 0 && <EmptyState title="ยังไม่มีรีวิว">เป็นคนแรกที่รีวิวร้านนี้</EmptyState>}
      {list.data?.items.map((rv) => (
        <article key={rv.id} className="flex flex-col gap-1 rounded-2xl border border-border bg-surface p-4">
          <div className="flex justify-between text-sm">
            <span className="font-semibold">
              {rv.author_name} <span className="font-normal text-star" aria-label={`${rv.score} จาก 5 ดาว`}>{"★".repeat(rv.score)}{"☆".repeat(5 - rv.score)}</span>
            </span>
            <span className="text-muted">{fmtShortDate(rv.created_at)}</span>
          </div>
          {rv.body && <p className="text-[15px]">{rv.body}</p>}
        </article>
      ))}
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-3">
          <Button disabled={page <= 1} onClick={() => setPage(page - 1)}>ก่อนหน้า</Button>
          <span className="text-sm tabular">{page} / {totalPages}</span>
          <Button disabled={page >= totalPages} onClick={() => setPage(page + 1)}>ถัดไป</Button>
        </div>
      )}
    </section>
  );
}

/** เขียนรีวิวใหม่ (POST) หรือแก้รีวิวเดิม (PUT) — ตัดสินจาก existing ที่โหลดจาก /reviews/mine */
function ReviewForm({ restaurantId, existing }: { restaurantId: string; existing: Review | null }) {
  const save = useSaveReview(restaurantId);
  const remove = useDeleteReview(restaurantId);
  const [done, setDone] = useState(false);
  const { register, handleSubmit, setValue, control, reset, formState } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { score: existing?.score ?? 0, body: existing?.body ?? "" },
  });
  useEffect(() => reset({ score: existing?.score ?? 0, body: existing?.body ?? "" }), [existing, reset]);
  const score = useWatch({ control, name: "score" });

  const onSubmit = handleSubmit(async (values) => {
    setDone(false);
    await save.mutateAsync({ exists: !!existing, input: values });
    setDone(true);
  });

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-3 rounded-2xl border border-border bg-surface p-4">
      <p className="font-semibold">{existing ? "รีวิวของคุณ" : "เขียนรีวิว"}</p>
      <div role="radiogroup" aria-label="คะแนน" className="flex gap-1">
        {[1, 2, 3, 4, 5].map((n) => (
          <button key={n} type="button" role="radio" aria-checked={score === n} aria-label={`${n} ดาว`}
            onClick={() => setValue("score", n, { shouldValidate: true })} className="grid size-11 place-items-center">
            <Star size={26} className={n <= score ? "fill-star text-star" : "text-soft"} aria-hidden />
          </button>
        ))}
      </div>
      {formState.errors.score && <p className="text-sm text-full">{formState.errors.score.message}</p>}
      <label className="flex flex-col gap-1 text-sm font-medium">
        ความเห็น
        <textarea {...register("body")} rows={3} className="rounded-xl border border-border-strong bg-surface p-3 text-base" />
      </label>
      {formState.errors.body && <p className="text-sm text-full">{formState.errors.body.message}</p>}
      {save.isError && <Alert tone="full" title={errorMessage(apiError(save.error))} />}
      {done && <Alert tone="ok" title="บันทึกรีวิวแล้ว" />}
      <div className="flex gap-2">
        <Button type="submit" disabled={save.isPending}>{existing ? "บันทึกการแก้ไข" : "ส่งรีวิว"}</Button>
        {existing && <Button variant="danger" disabled={remove.isPending} onClick={() => remove.mutate()}>ลบรีวิว</Button>}
      </div>
    </form>
  );
}
