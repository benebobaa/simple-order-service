-- +goose Up
CREATE TABLE products (
    id          UUID PRIMARY KEY,
    sku         TEXT NOT NULL UNIQUE CHECK (length(sku) BETWEEN 2 AND 64 AND sku = upper(btrim(sku))),
    name        TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    description TEXT NOT NULL DEFAULT '',
    price       BIGINT NOT NULL CHECK (price >= 0),
    stock       INTEGER NOT NULL CHECK (stock >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX products_created_at_idx ON products (created_at DESC, id DESC);

-- +goose Down
DROP TABLE products;
