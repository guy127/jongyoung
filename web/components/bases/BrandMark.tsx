/**
 * โลโก้จองยัง แบบ C — "จ" ในบับเบิลคำถาม ("จองยัง?") จาก mockup บอร์ด BrandIcon
 * ใช้ currentColor → สีตามตัวที่ครอบ (ขาวบนพื้น gradient, ขาวบน navbar owner)
 * ตัว "จ" เป็น path ที่แปลงมาจาก Kanit 600 แล้ว จึงหน้าตาเหมือนกันทุกที่โดยไม่ต้องรอโหลดฟอนต์
 * (path ชุดเดียวกันอยู่ใน app/icon.svg และธีม Keycloak — แก้ที่หนึ่งต้องแก้ให้ครบ)
 */
const BUBBLE = "M33 19 H67 A15 15 0 0 1 82 34 V59 A15 15 0 0 1 67 74 H45 L28 86 L31 73.4 A15 15 0 0 1 18 59 V34 A15 15 0 0 1 33 19 Z";
const JO_JAN =
  "M49.7 62.4Q48.7 62.4 47.5 62.2Q46.3 62 45.6 61.7L43.1 54.9L40.2 54.9L40.4 50.2L46.7 50.2L49.5 57.8Q51.4 57.7 52.5 56.2Q53.5 54.7 53.5 51.6Q53.5 49.4 52.9 48.1Q52.4 46.8 51 46.2Q49.6 45.7 47.2 45.7Q45.7 45.7 43.9 46Q42.2 46.2 40.6 46.7L40.6 42.4Q42.3 41.8 44.4 41.5Q46.6 41.1 48.7 41.1Q54.6 41.1 57.1 43.8Q59.6 46.5 59.6 51.6Q59.6 55.1 58.2 57.5Q56.8 59.9 54.6 61.2Q52.3 62.4 49.7 62.4Z";

export default function BrandMark({ size = 24, className }: { size?: number; className?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 100 100" aria-hidden className={className}>
      <path d={BUBBLE} fill="none" stroke="currentColor" strokeWidth={6} strokeLinejoin="round" />
      <path d={JO_JAN} fill="currentColor" />
    </svg>
  );
}
