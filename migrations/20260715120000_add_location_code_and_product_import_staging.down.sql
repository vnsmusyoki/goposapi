DROP INDEX IF EXISTS idx_product_import_batch_rows_batch_id;
DROP INDEX IF EXISTS idx_product_import_batches_business_id;
DROP TABLE IF EXISTS product_import_batch_rows;
DROP TABLE IF EXISTS product_import_batches;
DROP INDEX IF EXISTS idx_business_locations_business_location_code;
ALTER TABLE business_locations
    DROP COLUMN IF EXISTS location_code;
