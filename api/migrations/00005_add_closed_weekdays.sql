-- +goose Up
-- วันปิดประจำสัปดาห์ของร้าน: 7 บิต บิตที่ n = ปิดทุกวัน n ตาม time.Weekday ของ Go (0 = อาทิตย์ … 6 = เสาร์)
-- 0 = เปิดทุกวัน; 127 (ปิดครบ 7 วัน) ไม่อนุญาต — ร้านต้องเปิดอย่างน้อย 1 วัน
ALTER TABLE restaurants
    ADD COLUMN closed_weekdays smallint NOT NULL DEFAULT 0 CHECK (closed_weekdays BETWEEN 0 AND 126);

-- +goose Down
ALTER TABLE restaurants DROP COLUMN closed_weekdays;
