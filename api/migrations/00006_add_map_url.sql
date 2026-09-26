-- +goose Up
-- ลิงก์ Google Maps ที่เจ้าของร้านวางเอง (ไม่บังคับ) — ว่าง = หน้าเว็บค้นแผนที่จาก address แทน
ALTER TABLE restaurants ADD COLUMN map_url text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE restaurants DROP COLUMN map_url;
