DROP INDEX IF EXISTS idx_product_sub_categories_uuid_id;

ALTER TABLE product_sub_categories
    DROP COLUMN IF EXISTS uuid_id;
