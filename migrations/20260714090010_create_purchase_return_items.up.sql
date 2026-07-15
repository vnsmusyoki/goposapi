CREATE TABLE IF NOT EXISTS purchase_return_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_return_id UUID NOT NULL REFERENCES purchase_returns(id) ON DELETE CASCADE,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    product_name VARCHAR(255) NOT NULL DEFAULT '',
    sku VARCHAR(100) NOT NULL DEFAULT '',
    supplier_id UUID REFERENCES business_suppliers(id) ON DELETE SET NULL,
    supplier_name VARCHAR(255) NOT NULL DEFAULT '',
    location_id UUID NOT NULL REFERENCES business_locations(id) ON DELETE CASCADE,
    location_name VARCHAR(255) NOT NULL DEFAULT '',
    purchase_order_id UUID REFERENCES purchase_orders(id) ON DELETE SET NULL,
    inventory_batch_id UUID REFERENCES inventory_batches(id) ON DELETE SET NULL,
    lot_number VARCHAR(255) NOT NULL DEFAULT '',
    batch_number VARCHAR(255) NOT NULL DEFAULT '',
    expiry_date DATE,
    manufacture_date DATE,
    quantity NUMERIC(14,4) NOT NULL DEFAULT 0,
    unit_price NUMERIC(14,4) NOT NULL DEFAULT 0,
    line_total NUMERIC(14,4) NOT NULL DEFAULT 0,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT purchase_return_items_non_negative
        CHECK (
            quantity >= 0
            AND unit_price >= 0
            AND line_total >= 0
        )
);

CREATE TRIGGER set_purchase_return_items_updated_at
BEFORE UPDATE ON purchase_return_items
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_purchase_return_items_purchase_return_id
    ON purchase_return_items (purchase_return_id);

CREATE INDEX IF NOT EXISTS idx_purchase_return_items_business_id
    ON purchase_return_items (business_id);

CREATE INDEX IF NOT EXISTS idx_purchase_return_items_product_id
    ON purchase_return_items (product_id);

CREATE INDEX IF NOT EXISTS idx_purchase_return_items_inventory_batch_id
    ON purchase_return_items (inventory_batch_id);

CREATE INDEX IF NOT EXISTS idx_purchase_return_items_deleted_at
    ON purchase_return_items (deleted_at);
