ALTER TABLE product_sub_categories
    ADD COLUMN IF NOT EXISTS uuid_id UUID NOT NULL DEFAULT gen_random_uuid();

UPDATE product_sub_categories
SET uuid_id = COALESCE(uuid_id, gen_random_uuid())
WHERE uuid_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_sub_categories_uuid_id
    ON product_sub_categories (uuid_id);
