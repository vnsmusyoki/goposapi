ALTER TABLE product_categories
    ADD COLUMN IF NOT EXISTS deleted BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS deleted_by UUID;

ALTER TABLE product_sub_categories
    ADD COLUMN IF NOT EXISTS deleted BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS deleted_by UUID;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'product_categories_category_code_key'
    ) THEN
        ALTER TABLE product_categories DROP CONSTRAINT product_categories_category_code_key;
    END IF;
END $$;

DROP INDEX IF EXISTS idx_product_categories_business_category_code_active;
CREATE UNIQUE INDEX IF NOT EXISTS idx_product_categories_business_category_code_active
    ON product_categories (business_id, category_code)
    WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS idx_product_sub_categories_business_code;
CREATE UNIQUE INDEX IF NOT EXISTS idx_product_sub_categories_business_code
    ON product_sub_categories (business_id, sub_category_code)
    WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS idx_product_sub_categories_business_parent_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_product_sub_categories_business_parent_name
    ON product_sub_categories (business_id, parent_category_id, LOWER(name))
    WHERE deleted_at IS NULL;
