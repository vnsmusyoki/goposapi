DROP INDEX IF EXISTS idx_product_sub_categories_business_parent_name;
DROP INDEX IF EXISTS idx_product_sub_categories_business_code;
DROP INDEX IF EXISTS idx_product_categories_business_category_code_active;

ALTER TABLE product_categories
    DROP COLUMN IF EXISTS deleted_by,
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS deleted;

ALTER TABLE product_sub_categories
    DROP COLUMN IF EXISTS deleted_by,
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS deleted;

ALTER TABLE product_categories
    ADD CONSTRAINT product_categories_category_code_key UNIQUE (category_code);
