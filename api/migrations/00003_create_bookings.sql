-- +goose Up
CREATE TABLE bookings (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    restaurant_id uuid        NOT NULL REFERENCES restaurants (id),
    user_id       uuid        NOT NULL REFERENCES users (id),
    party_size    int         NOT NULL CHECK (party_size > 0),
    start_at      timestamptz NOT NULL,
    end_at        timestamptz NOT NULL,
    status        text        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'cancelled')),
    cancelled_at  timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CHECK (end_at > start_at)
);

-- ใช้ตอนหา booking ที่ทับช่วงเวลาของร้าน (นับที่นั่ง) — เฉพาะที่ยัง active
CREATE INDEX bookings_restaurant_time_active_idx ON bookings (restaurant_id, start_at, end_at) WHERE status = 'active';
-- ใช้ในหน้า "การจองของฉัน"
CREATE INDEX bookings_user_start_idx ON bookings (user_id, start_at DESC);

-- +goose Down
DROP TABLE bookings;
