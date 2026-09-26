-- +goose Up
-- ช่วงที่ร้านปิดชั่วคราว (ไฟดับ, ปิดปรับปรุง, หยุดยาว) เป็นช่วงเวลาจริง [start_at, end_at)
-- ปิดทั้งวัน/หลายวันก็เก็บรูปแบบเดียวกัน (= ช่วงที่ครอบทั้งรอบของวันนั้น)
CREATE TABLE restaurant_closures (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    restaurant_id uuid NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    start_at      timestamptz NOT NULL,
    end_at        timestamptz NOT NULL,
    reason        text NOT NULL CHECK (reason <> ''),
    created_at    timestamptz NOT NULL DEFAULT now(),
    CHECK (end_at > start_at)
);
CREATE INDEX restaurant_closures_restaurant_end_idx ON restaurant_closures (restaurant_id, end_at);

-- +goose Down
DROP TABLE restaurant_closures;
