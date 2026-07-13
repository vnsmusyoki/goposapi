CREATE TABLE IF NOT EXISTS sales (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES business_locations(id) ON DELETE RESTRICT,
    reference_number VARCHAR(100) NOT NULL DEFAULT '',
    sale_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    customer_name VARCHAR(255) NOT NULL DEFAULT '',
    customer_phone VARCHAR(50) NOT NULL DEFAULT '',
    customer_email VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(30) NOT NULL DEFAULT 'completed',
    subtotal NUMERIC(14,4) NOT NULL DEFAULT 0,
    total_discount NUMERIC(14,4) NOT NULL DEFAULT 0,
    total_tax NUMERIC(14,4) NOT NULL DEFAULT 0,
    grand_total NUMERIC(14,4) NOT NULL DEFAULT 0,
    items_count INTEGER NOT NULL DEFAULT 0,
    total_quantity NUMERIC(14,4) NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT sales_numeric_non_negative
        CHECK (
            subtotal >= 0
            AND total_discount >= 0
            AND total_tax >= 0
            AND grand_total >= 0
            AND items_count >= 0
            AND total_quantity >= 0
        )
);

CREATE TRIGGER set_sales_updated_at
BEFORE UPDATE ON sales
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_sales_business_id
    ON sales (business_id);

CREATE INDEX IF NOT EXISTS idx_sales_business_created_at
    ON sales (business_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_sales_deleted_at
    ON sales (deleted_at);

CREATE INDEX IF NOT EXISTS idx_sales_location_id
    ON sales (location_id);
