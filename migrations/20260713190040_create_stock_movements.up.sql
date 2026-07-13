CREATE TABLE IF NOT EXISTS stock_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    location_id UUID NOT NULL REFERENCES business_locations(id) ON DELETE CASCADE,
    inventory_balance_id UUID REFERENCES inventory_balances(id) ON DELETE SET NULL,
    inventory_batch_id UUID REFERENCES inventory_batches(id) ON DELETE SET NULL,
    movement_type VARCHAR(50) NOT NULL REFERENCES inventory_movement_types(movement_type) ON UPDATE CASCADE ON DELETE RESTRICT,
    source_type VARCHAR(50) NOT NULL DEFAULT '',
    source_id UUID,
    reference_number VARCHAR(100) NOT NULL DEFAULT '',
    quantity_in NUMERIC(14,4) NOT NULL DEFAULT 0,
    quantity_out NUMERIC(14,4) NOT NULL DEFAULT 0,
    unit_cost NUMERIC(14,4) NOT NULL DEFAULT 0,
    stock_before NUMERIC(14,4) NOT NULL DEFAULT 0,
    stock_after NUMERIC(14,4) NOT NULL DEFAULT 0,
    note TEXT,
    performed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT stock_movements_non_negative
        CHECK (
            quantity_in >= 0
            AND quantity_out >= 0
            AND unit_cost >= 0
            AND stock_before >= 0
            AND stock_after >= 0
        ),
    CONSTRAINT stock_movements_direction_check
        CHECK (
            (quantity_in = 0 OR quantity_out = 0)
            AND (quantity_in > 0 OR quantity_out > 0)
        )
);

CREATE INDEX IF NOT EXISTS idx_stock_movements_business_id
    ON stock_movements (business_id);

CREATE INDEX IF NOT EXISTS idx_stock_movements_product_id
    ON stock_movements (product_id);

CREATE INDEX IF NOT EXISTS idx_stock_movements_location_id
    ON stock_movements (location_id);

CREATE INDEX IF NOT EXISTS idx_stock_movements_batch_id
    ON stock_movements (inventory_batch_id);

CREATE INDEX IF NOT EXISTS idx_stock_movements_balance_id
    ON stock_movements (inventory_balance_id);

CREATE INDEX IF NOT EXISTS idx_stock_movements_source
    ON stock_movements (source_type, source_id);

CREATE INDEX IF NOT EXISTS idx_stock_movements_occurred_at
    ON stock_movements (business_id, occurred_at DESC);
