-- +goose Up
CREATE TABLE users (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    keycloak_uid text        NOT NULL, -- claims.sub (ไม่เปลี่ยนตลอดชีวิตบัญชี)
    email        text        NOT NULL,
    display_name text        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz
);

-- partial unique index: คนที่ถูก soft delete แล้วล็อกอินใหม่จะไม่ชนกับแถวเก่า
CREATE UNIQUE INDEX users_keycloak_uid_active_key ON users (keycloak_uid) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE users;
