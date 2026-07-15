ALTER TABLE product_import_batch_rows
    DROP CONSTRAINT IF EXISTS product_import_batch_rows_sub_category_id_fkey;

ALTER TABLE product_import_batch_rows
    DROP COLUMN IF EXISTS sub_category_id;

ALTER TABLE product_import_batch_rows
    ADD COLUMN sub_category_id UUID NULL;

ALTER TABLE product_import_batch_rows
    ADD CONSTRAINT product_import_batch_rows_sub_category_id_fkey
        FOREIGN KEY (sub_category_id)
        REFERENCES product_sub_categories(uuid_id)
        ON DELETE SET NULL;
