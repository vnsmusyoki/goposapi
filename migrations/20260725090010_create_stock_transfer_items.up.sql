CREATE TABLE IF NOT EXISTS stock_transfer_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stock_transfer_id UUID NOT NULL REFERENCES stock_transfers(id) ON DELETE CASCADE,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    variation_id UUID REFERENCES product_variants(id) ON DELETE SET NULL,
    product_name VARCHAR(255) NOT NULL,
    sku VARCHAR(255) NOT NULL DEFAULT '',
    unit VARCHAR(100) NOT NULL DEFAULT '',
    quantity NUMERIC(14,4) NOT NULL DEFAULT 1,
    unit_cost NUMERIC(14,4) NOT NULL DEFAULT 0,
    line_total NUMERIC(14,4) NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT stock_transfer_items_numeric_non_negative
        CHECK (
            quantity > 0
            AND unit_cost >= 0
            AND line_total >= 0
        )
);

CREATE TRIGGER set_stock_transfer_items_updated_at
BEFORE UPDATE ON stock_transfer_items
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_stock_transfer_items_stock_transfer_id
    ON stock_transfer_items (stock_transfer_id);

CREATE INDEX IF NOT EXISTS idx_stock_transfer_items_business_id
    ON stock_transfer_items (business_id);

CREATE INDEX IF NOT EXISTS idx_stock_transfer_items_product_id
    ON stock_transfer_items (product_id);

CREATE INDEX IF NOT EXISTS idx_stock_transfer_items_variation_id
    ON stock_transfer_items (variation_id);
