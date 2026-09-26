"use client";

import { Alert } from "@/components/bases/ui";
import { closureRangeLabel } from "@/lib/format";
import { useClosures } from "@/services/closures";

/** แถบแจ้งบนหน้าร้านเมื่อมีช่วงปิดชั่วคราวที่ยังไม่จบ — ให้เห็นก่อนเลือกเวลา */
export default function ClosureBanner({ restaurantId }: { restaurantId: string }) {
  const closures = useClosures(restaurantId);
  if (!closures.data?.length) return null;
  return (
    <Alert tone="warn" title="ร้านปิดชั่วคราว">
      <ul className="flex flex-col gap-0.5">
        {closures.data.map((c) => <li key={c.id} className="tabular">{closureRangeLabel(c)} · {c.reason}</li>)}
      </ul>
    </Alert>
  );
}
