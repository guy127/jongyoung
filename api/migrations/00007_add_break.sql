-- +goose Up
-- ช่วงพักร้านภายในรอบเปิด (เช่น 11:00–22:00 พัก 14:00–17:00) เป็นนาทีจากเที่ยงคืนเวลาไทย
-- start == end (ค่าเริ่มต้น 0/0) = ไม่มีช่วงพัก
-- เงื่อนไข "อยู่ข้างในรอบ" ตรวจที่ Go (Hours.ValidBreak) เพราะต้องรู้ความยาวรอบซึ่งขึ้นกับเปิดข้ามคืน/24 ชม.
ALTER TABLE restaurants
    ADD COLUMN break_start_minute int NOT NULL DEFAULT 0 CHECK (break_start_minute BETWEEN 0 AND 1439),
    ADD COLUMN break_end_minute   int NOT NULL DEFAULT 0 CHECK (break_end_minute BETWEEN 0 AND 1439);

-- +goose Down
ALTER TABLE restaurants DROP COLUMN break_start_minute, DROP COLUMN break_end_minute;
