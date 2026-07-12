CREATE TABLE IF NOT EXISTS product_price_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    buying_price NUMERIC(14,4) NOT NULL DEFAULT 0,
    selling_price NUMERIC(14,4) NOT NULL DEFAULT 0,
    reason TEXT,
    changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_product_price_history_business_id
    ON product_price_history (business_id);

CREATE INDEX IF NOT EXISTS idx_product_price_history_product_id_created_at
    ON product_price_history (product_id, created_at DESC);
