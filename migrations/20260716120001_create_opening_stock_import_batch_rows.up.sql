CREATE TABLE IF NOT EXISTS opening_stock_import_batch_rows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id UUID NOT NULL REFERENCES opening_stock_import_batches(id) ON DELETE CASCADE,
    row_number INTEGER NOT NULL,
    sku VARCHAR(100) NOT NULL DEFAULT '',
    product_id UUID REFERENCES products(id) ON DELETE SET NULL,
    location_id UUID REFERENCES business_locations(id) ON DELETE SET NULL,
    quantity VARCHAR(50) NOT NULL DEFAULT '',
    unit_cost_before_tax VARCHAR(50) NOT NULL DEFAULT '',
    lot_number VARCHAR(255) NOT NULL DEFAULT '',
    expiry_date VARCHAR(50) NOT NULL DEFAULT '',
    row_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    validation_errors JSONB NOT NULL DEFAULT '[]'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    imported_inventory_batch_id UUID REFERENCES inventory_batches(id) ON DELETE SET NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_opening_stock_import_batch_rows_batch_id
    ON opening_stock_import_batch_rows (batch_id, row_number ASC);
