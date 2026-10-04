-- +goose Up
CREATE TABLE users (
    id            UUID PRIMARY KEY,
    name          TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    email         TEXT NOT NULL UNIQUE CHECK (length(btrim(email)) > 0),
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE users;
