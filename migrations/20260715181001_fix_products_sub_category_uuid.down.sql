ALTER TABLE products
    DROP CONSTRAINT IF EXISTS products_sub_category_id_fkey;

ALTER TABLE products
    DROP COLUMN IF EXISTS sub_category_id;

ALTER TABLE products
    ADD COLUMN sub_category_id INTEGER NULL;

ALTER TABLE products
    ADD CONSTRAINT products_sub_category_id_fkey
        FOREIGN KEY (sub_category_id)
        REFERENCES product_sub_categories(id)
        ON DELETE SET NULL;
