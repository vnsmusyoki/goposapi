CREATE TABLE IF NOT EXISTS sale_item_batch_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sale_id UUID NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
    sale_item_id UUID NOT NULL REFERENCES sale_items(id) ON DELETE CASCADE,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    inventory_batch_id UUID NOT NULL REFERENCES inventory_batches(id) ON DELETE RESTRICT,
    allocated_quantity NUMERIC(14,4) NOT NULL DEFAULT 0,
    unit_cost NUMERIC(14,4) NOT NULL DEFAULT 0,
    line_total NUMERIC(14,4) NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT sale_item_batch_allocations_numeric_non_negative
        CHECK (
            allocated_quantity > 0
            AND unit_cost >= 0
            AND line_total >= 0
        )
);

CREATE TRIGGER set_sale_item_batch_allocations_updated_at
BEFORE UPDATE ON sale_item_batch_allocations
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_sale_item_batch_allocations_sale_id
    ON sale_item_batch_allocations (sale_id);

CREATE INDEX IF NOT EXISTS idx_sale_item_batch_allocations_sale_item_id
    ON sale_item_batch_allocations (sale_item_id);

CREATE INDEX IF NOT EXISTS idx_sale_item_batch_allocations_business_id
    ON sale_item_batch_allocations (business_id);

CREATE INDEX IF NOT EXISTS idx_sale_item_batch_allocations_inventory_batch_id
    ON sale_item_batch_allocations (inventory_batch_id);
