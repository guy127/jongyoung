-- +goose Up
CREATE TABLE reviews (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    restaurant_id uuid        NOT NULL REFERENCES restaurants (id),
    user_id       uuid        NOT NULL REFERENCES users (id),
    score         int         NOT NULL CHECK (score BETWEEN 1 AND 5),
    body          text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (restaurant_id, user_id) -- 1 คน 1 รีวิวต่อร้าน
);

CREATE INDEX reviews_restaurant_created_idx ON reviews (restaurant_id, created_at DESC);

-- +goose Down
DROP TABLE reviews;
