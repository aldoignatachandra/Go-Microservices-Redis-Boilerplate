-- +migrate Up
-- Alter products table: add owner_id, has_variant, images; remove description, status, category_id
ALTER TABLE products ADD COLUMN IF NOT EXISTS owner_id UUID NOT NULL;
ALTER TABLE products ADD COLUMN IF NOT EXISTS has_variant BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE products ADD COLUMN IF NOT EXISTS images TEXT;

ALTER TABLE products DROP COLUMN IF EXISTS description;
ALTER TABLE products DROP COLUMN IF EXISTS status;
ALTER TABLE products DROP COLUMN IF EXISTS category_id;

-- Create indexes
CREATE INDEX IF NOT EXISTS products_owner_id_idx ON products(owner_id);
CREATE INDEX IF NOT EXISTS products_has_variant_idx ON products(has_variant);

-- +migrate Down
-- Reverse: remove added columns, add back removed columns
ALTER TABLE products DROP COLUMN IF EXISTS owner_id;
ALTER TABLE products DROP COLUMN IF EXISTS has_variant;
ALTER TABLE products DROP COLUMN IF EXISTS images;

ALTER TABLE products ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE products ADD COLUMN IF NOT EXISTS status VARCHAR(50);
ALTER TABLE products ADD COLUMN IF NOT EXISTS category_id UUID NOT NULL;

DROP INDEX IF EXISTS products_owner_id_idx;
DROP INDEX IF EXISTS products_has_variant_idx;
