CREATE TABLE IF NOT EXISTS sale_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sale_id UUID NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    product_name VARCHAR(255) NOT NULL,
    sku VARCHAR(255) NOT NULL DEFAULT '',
    unit VARCHAR(100) NOT NULL DEFAULT '',
    quantity NUMERIC(14,4) NOT NULL DEFAULT 1,
    unit_cost NUMERIC(14,4) NOT NULL DEFAULT 0,
    discount_percentage NUMERIC(10,2) NOT NULL DEFAULT 0,
    discount_amount NUMERIC(14,4) NOT NULL DEFAULT 0,
    tax_rate NUMERIC(10,2) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(14,4) NOT NULL DEFAULT 0,
    unit_price NUMERIC(14,4) NOT NULL DEFAULT 0,
    line_total NUMERIC(14,4) NOT NULL DEFAULT 0,
    batch_tracking_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT sale_items_numeric_non_negative
        CHECK (
            quantity > 0
            AND unit_cost >= 0
            AND discount_percentage >= 0
            AND discount_amount >= 0
            AND tax_rate >= 0
            AND tax_amount >= 0
            AND unit_price >= 0
            AND line_total >= 0
        )
);

CREATE TRIGGER set_sale_items_updated_at
BEFORE UPDATE ON sale_items
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_sale_items_sale_id
    ON sale_items (sale_id);

CREATE INDEX IF NOT EXISTS idx_sale_items_business_id
    ON sale_items (business_id);

CREATE INDEX IF NOT EXISTS idx_sale_items_deleted_at
    ON sale_items (deleted_at);
