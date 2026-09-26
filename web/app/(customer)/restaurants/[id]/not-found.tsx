import { EmptyState, LinkButton } from "@/components/bases/ui";

/** page.tsx เรียก notFound() เมื่อ API ตอบ 404 — ร้านไม่มีอยู่ ถูกลบไปแล้ว หรือเป็นลิงก์เก่า */
export default function RestaurantNotFound() {
  return (
    <EmptyState title="ไม่พบร้านนี้" action={<LinkButton href="/" variant="cta">ค้นหาร้านอื่น</LinkButton>}>
      ร้านอาจปิดรับจองไปแล้ว หรือลิงก์นี้เก่าเกินไป — ลองค้นหาร้านที่ว่างในเวลาที่คุณอยากไปแทน
    </EmptyState>
  );
}
