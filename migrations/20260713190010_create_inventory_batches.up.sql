CREATE TABLE IF NOT EXISTS inventory_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    location_id UUID NOT NULL REFERENCES business_locations(id) ON DELETE CASCADE,
    source_type VARCHAR(50) NOT NULL DEFAULT '',
    source_id UUID,
    lot_number VARCHAR(255) NOT NULL DEFAULT '',
    batch_number VARCHAR(255) NOT NULL DEFAULT '',
    expiry_date DATE,
    unit_cost NUMERIC(14,4) NOT NULL DEFAULT 0,
    quantity_received NUMERIC(14,4) NOT NULL DEFAULT 0,
    quantity_remaining NUMERIC(14,4) NOT NULL DEFAULT 0,
    received_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT inventory_batches_non_negative
        CHECK (
            unit_cost >= 0
            AND quantity_received >= 0
            AND quantity_remaining >= 0
        )
);

CREATE TRIGGER set_inventory_batches_updated_at
BEFORE UPDATE ON inventory_batches
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_inventory_batches_business_id
    ON inventory_batches (business_id);

CREATE INDEX IF NOT EXISTS idx_inventory_batches_product_id
    ON inventory_batches (product_id);

CREATE INDEX IF NOT EXISTS idx_inventory_batches_location_id
    ON inventory_batches (location_id);

CREATE INDEX IF NOT EXISTS idx_inventory_batches_expiry_date
    ON inventory_batches (expiry_date);

CREATE INDEX IF NOT EXISTS idx_inventory_batches_source
    ON inventory_batches (source_type, source_id);

CREATE INDEX IF NOT EXISTS idx_inventory_batches_remaining
    ON inventory_batches (business_id, product_id, location_id, quantity_remaining);
