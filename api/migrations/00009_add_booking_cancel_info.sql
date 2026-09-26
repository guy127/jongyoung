-- +goose Up
-- ใครเป็นคนยกเลิก: ลูกค้ายกเลิกเอง หรือร้านยกเลิก (ปิดชั่วคราว/ลบร้าน) พร้อมเหตุผลที่ลูกค้าจะเห็น
-- การจองที่ยกเลิกไปก่อนมีคอลัมน์นี้คงเป็น null (ไม่ backfill เพราะไม่รู้จริงว่าใครยกเลิก)
ALTER TABLE bookings
    ADD COLUMN cancelled_by  text CHECK (cancelled_by IN ('customer', 'restaurant')),
    ADD COLUMN cancel_reason text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE bookings DROP COLUMN cancelled_by, DROP COLUMN cancel_reason;
