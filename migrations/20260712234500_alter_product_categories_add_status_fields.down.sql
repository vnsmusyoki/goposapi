ALTER TABLE product_categories
    DROP COLUMN IF EXISTS sort_order,
    DROP COLUMN IF EXISTS featured,
    DROP COLUMN IF EXISTS active;
