CREATE TABLE IF NOT EXISTS inventory_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    location_id UUID NOT NULL REFERENCES business_locations(id) ON DELETE CASCADE,
    quantity_available NUMERIC(14,4) NOT NULL DEFAULT 0,
    quantity_reserved NUMERIC(14,4) NOT NULL DEFAULT 0,
    last_movement_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT inventory_balances_non_negative
        CHECK (quantity_available >= 0 AND quantity_reserved >= 0),
    CONSTRAINT inventory_balances_unique_active
        UNIQUE (business_id, product_id, location_id)
);

CREATE TRIGGER set_inventory_balances_updated_at
BEFORE UPDATE ON inventory_balances
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_inventory_balances_business_id
    ON inventory_balances (business_id);

CREATE INDEX IF NOT EXISTS idx_inventory_balances_product_id
    ON inventory_balances (product_id);

CREATE INDEX IF NOT EXISTS idx_inventory_balances_location_id
    ON inventory_balances (location_id);

CREATE INDEX IF NOT EXISTS idx_inventory_balances_business_location
    ON inventory_balances (business_id, location_id);
