-- +migrate Up
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    stock INTEGER NOT NULL DEFAULT 0,
    owner_id UUID NOT NULL,
    has_variant BOOLEAN NOT NULL DEFAULT FALSE,
    images TEXT
);

CREATE INDEX IF NOT EXISTS products_owner_id_idx ON products(owner_id);
CREATE INDEX IF NOT EXISTS products_has_variant_idx ON products(has_variant);
CREATE INDEX IF NOT EXISTS products_deleted_at_idx ON products(deleted_at);

-- +migrate Down
DROP TABLE IF EXISTS products;
