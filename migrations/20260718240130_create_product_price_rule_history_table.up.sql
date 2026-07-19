CREATE TABLE IF NOT EXISTS product_price_rule_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    product_price_id UUID NULL REFERENCES product_prices(id) ON DELETE SET NULL,
    action VARCHAR(30) NOT NULL,
    price_type VARCHAR(50) NOT NULL,
    min_quantity NUMERIC(14,4) NOT NULL DEFAULT 1,
    old_price NUMERIC(14,4) NULL,
    new_price NUMERIC(14,4) NOT NULL,
    location_id UUID NULL REFERENCES business_locations(id) ON DELETE SET NULL,
    customer_group VARCHAR(150) NULL,
    starts_at TIMESTAMPTZ NULL,
    ends_at TIMESTAMPTZ NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    priority INTEGER NOT NULL DEFAULT 100,
    reason TEXT NULL,
    changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT product_price_rule_history_action_check
        CHECK (action IN ('created', 'updated', 'deleted', 'deactivated', 'reactivated')),
    CONSTRAINT product_price_rule_history_min_quantity_check
        CHECK (min_quantity > 0),
    CONSTRAINT product_price_rule_history_new_price_check
        CHECK (new_price >= 0),
    CONSTRAINT product_price_rule_history_old_price_check
        CHECK (old_price IS NULL OR old_price >= 0)
);

CREATE INDEX IF NOT EXISTS idx_product_price_rule_history_business_id
    ON product_price_rule_history (business_id);

CREATE INDEX IF NOT EXISTS idx_product_price_rule_history_product_created_at
    ON product_price_rule_history (product_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_product_price_rule_history_product_price_id
    ON product_price_rule_history (product_price_id)
    WHERE product_price_id IS NOT NULL;
