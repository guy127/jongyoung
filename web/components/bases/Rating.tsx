import { Star } from "lucide-react";

import { ratingLabel } from "@/lib/format";
import type { Rating as RatingT } from "@/lib/types";

/** รีวิว ≥ 5 → ดาว + ตัวเลข; 1–4 → ป้าย "รีวิวน้อย"; 0 → "ยังไม่มีรีวิว" */
export default function Rating({ rating }: { rating: RatingT }) {
  const label = ratingLabel(rating);
  if (label.kind === "stars") {
    return (
      <span className="inline-flex items-center gap-1 tabular">
        <Star size={14} className="fill-star text-star" aria-hidden />
        <span className="font-semibold text-text">{label.text.split(" · ")[0]}</span>
        <span>· {label.text.split(" · ")[1]}</span>
      </span>
    );
  }
  if (label.kind === "few") return <span className="rounded-full bg-chip px-2 text-[13px] text-muted">{label.text}</span>;
  return <span>{label.text}</span>;
}
