-- +goose Up
-- แจ้งเตือนในเว็บ — เก็บ snapshot ของการจองตอนเกิดเหตุ (การจองอาจถูกแก้ภายหลัง แต่แจ้งเตือนต้องบอกสิ่งที่เกิดตอนนั้น)
-- ไม่เก็บข้อความ: หน้าเว็บประกอบข้อความไทยจาก kind เอง
CREATE TABLE notifications (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL REFERENCES users(id),
    kind            text NOT NULL CHECK (kind IN ('booking_created', 'booking_updated', 'booking_cancelled', 'booking_cancelled_by_restaurant')),
    booking_id      uuid NOT NULL REFERENCES bookings(id),
    restaurant_id   uuid NOT NULL REFERENCES restaurants(id),
    restaurant_name text NOT NULL,
    customer_name   text NOT NULL,
    business_date   date NOT NULL,
    start_at        timestamptz NOT NULL,
    end_at          timestamptz NOT NULL,
    party_size      int NOT NULL,
    reason          text NOT NULL DEFAULT '',
    read_at         timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX notifications_user_created_idx ON notifications (user_id, created_at DESC);

-- +goose Down
DROP TABLE notifications;
