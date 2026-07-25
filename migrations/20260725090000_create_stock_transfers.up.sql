CREATE TABLE IF NOT EXISTS stock_transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    reference_number VARCHAR(100) NOT NULL DEFAULT '',
    transfer_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(30) NOT NULL DEFAULT 'draft',
    transfer_type VARCHAR(30) NOT NULL DEFAULT 'local',
    source_location_id UUID NOT NULL REFERENCES business_locations(id) ON DELETE RESTRICT,
    destination_location_id UUID NOT NULL REFERENCES business_locations(id) ON DELETE RESTRICT,
    currency_code VARCHAR(20) NOT NULL DEFAULT 'USD',
    shipping_charges NUMERIC(14,4) NOT NULL DEFAULT 0,
    subtotal NUMERIC(14,4) NOT NULL DEFAULT 0,
    total_amount NUMERIC(14,4) NOT NULL DEFAULT 0,
    items_count INTEGER NOT NULL DEFAULT 0,
    total_quantity NUMERIC(14,4) NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    CONSTRAINT stock_transfers_numeric_non_negative
        CHECK (
            shipping_charges >= 0
            AND subtotal >= 0
            AND total_amount >= 0
            AND items_count >= 0
            AND total_quantity >= 0
        ),
    CONSTRAINT stock_transfers_locations_check
        CHECK (source_location_id <> destination_location_id)
);

CREATE TRIGGER set_stock_transfers_updated_at
BEFORE UPDATE ON stock_transfers
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_stock_transfers_business_id
    ON stock_transfers (business_id);

CREATE INDEX IF NOT EXISTS idx_stock_transfers_business_transfer_date
    ON stock_transfers (business_id, transfer_date DESC);

CREATE INDEX IF NOT EXISTS idx_stock_transfers_status
    ON stock_transfers (status);

CREATE INDEX IF NOT EXISTS idx_stock_transfers_source_location_id
    ON stock_transfers (source_location_id);

CREATE INDEX IF NOT EXISTS idx_stock_transfers_destination_location_id
    ON stock_transfers (destination_location_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_stock_transfers_business_reference_active
    ON stock_transfers (business_id, LOWER(reference_number))
    WHERE deleted_at IS NULL AND reference_number <> '';
