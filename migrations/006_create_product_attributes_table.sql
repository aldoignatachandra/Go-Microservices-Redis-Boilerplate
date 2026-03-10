-- +migrate Up
CREATE TABLE IF NOT EXISTS product_attributes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    "values" JSONB NOT NULL,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS product_attributes_product_id_idx ON product_attributes(product_id);
CREATE INDEX IF NOT EXISTS product_attributes_product_id_name_idx ON product_attributes(product_id, name);
CREATE INDEX IF NOT EXISTS product_attributes_deleted_at_idx ON product_attributes(deleted_at);

-- +migrate Down
DROP TABLE IF EXISTS product_attributes;
