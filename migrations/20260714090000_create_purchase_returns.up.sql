CREATE TABLE IF NOT EXISTS purchase_returns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    parent_purchase_id UUID REFERENCES purchase_orders(id) ON DELETE SET NULL,
    parent_purchase_reference VARCHAR(100) NOT NULL DEFAULT '',
    reference_number VARCHAR(100) NOT NULL,
    return_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    location_id UUID NOT NULL REFERENCES business_locations(id) ON DELETE CASCADE,
    supplier_id UUID REFERENCES business_suppliers(id) ON DELETE SET NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'returned',
    payment_status VARCHAR(30) NOT NULL DEFAULT 'unpaid',
    grand_total NUMERIC(14,4) NOT NULL DEFAULT 0,
    payment_due NUMERIC(14,4) NOT NULL DEFAULT 0,
    total_quantity NUMERIC(14,4) NOT NULL DEFAULT 0,
    items_count INTEGER NOT NULL DEFAULT 0,
    return_reason TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT purchase_returns_non_negative
        CHECK (
            grand_total >= 0
            AND payment_due >= 0
            AND total_quantity >= 0
            AND items_count >= 0
        ),
    CONSTRAINT purchase_returns_unique_reference
        UNIQUE (business_id, reference_number)
);

CREATE TRIGGER set_purchase_returns_updated_at
BEFORE UPDATE ON purchase_returns
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_purchase_returns_business_id
    ON purchase_returns (business_id);

CREATE INDEX IF NOT EXISTS idx_purchase_returns_parent_purchase_id
    ON purchase_returns (parent_purchase_id);

CREATE INDEX IF NOT EXISTS idx_purchase_returns_location_id
    ON purchase_returns (location_id);

CREATE INDEX IF NOT EXISTS idx_purchase_returns_supplier_id
    ON purchase_returns (supplier_id);

CREATE INDEX IF NOT EXISTS idx_purchase_returns_return_date
    ON purchase_returns (business_id, return_date DESC);

CREATE INDEX IF NOT EXISTS idx_purchase_returns_deleted_at
    ON purchase_returns (deleted_at);
