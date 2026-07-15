ALTER TABLE business_locations
    ADD COLUMN IF NOT EXISTS location_code VARCHAR(50);

UPDATE business_locations
SET location_code = location_id
WHERE location_code IS NULL
  AND location_id IS NOT NULL
  AND TRIM(location_id) <> '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_business_locations_business_location_code
    ON business_locations (business_id, LOWER(location_code))
    WHERE location_code IS NOT NULL;

CREATE TABLE IF NOT EXISTS product_import_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS product_import_batch_rows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id UUID NOT NULL REFERENCES product_import_batches(id) ON DELETE CASCADE,
    row_number INTEGER NOT NULL,
    row_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    validation_errors JSONB NOT NULL DEFAULT '[]'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    imported_product_id UUID REFERENCES products(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_product_import_batches_business_id
    ON product_import_batches (business_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_product_import_batch_rows_batch_id
    ON product_import_batch_rows (batch_id, row_number ASC);
