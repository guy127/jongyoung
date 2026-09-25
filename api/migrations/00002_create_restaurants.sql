-- +goose Up
CREATE TABLE restaurants (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id              uuid        NOT NULL REFERENCES users (id),
    name                  text        NOT NULL,
    description           text,
    cuisine               text,
    address               text        NOT NULL,
    seats                 int         NOT NULL CHECK (seats > 0),
    -- นาทีนับจากเที่ยงคืนเวลาไทย; close <= open = ข้ามเที่ยงคืน, open = close = 24 ชม.
    open_minute           int         NOT NULL CHECK (open_minute BETWEEN 0 AND 1439),
    close_minute          int         NOT NULL CHECK (close_minute BETWEEN 0 AND 1439),
    cancel_before_minutes int         NOT NULL DEFAULT 30 CHECK (cancel_before_minutes >= 30),
    rating_sum            int         NOT NULL DEFAULT 0 CHECK (rating_sum >= 0),
    rating_count          int         NOT NULL DEFAULT 0 CHECK (rating_count >= 0),
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    deleted_at            timestamptz
);

CREATE INDEX restaurants_owner_id_idx ON restaurants (owner_id);
CREATE INDEX restaurants_created_at_idx ON restaurants (created_at DESC, id) WHERE deleted_at IS NULL;

CREATE TABLE restaurant_images (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    restaurant_id uuid NOT NULL REFERENCES restaurants (id) ON DELETE CASCADE,
    url           text NOT NULL,
    sort_order    int  NOT NULL DEFAULT 0 -- 0 = รูปปก
);

CREATE INDEX restaurant_images_restaurant_id_idx ON restaurant_images (restaurant_id, sort_order);

-- +goose Down
DROP TABLE restaurant_images;
DROP TABLE restaurants;
