CREATE TABLE IF NOT EXISTS product_prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    location_id UUID NULL REFERENCES business_locations(id) ON DELETE CASCADE,
    customer_group VARCHAR(150) NULL,
    price_type VARCHAR(50) NOT NULL DEFAULT 'retail',
    min_quantity NUMERIC(14,4) NOT NULL DEFAULT 1,
    price NUMERIC(14,4) NOT NULL,
    starts_at TIMESTAMPTZ NULL,
    ends_at TIMESTAMPTZ NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    priority INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT product_prices_price_type_check
        CHECK (price_type IN ('retail', 'wholesale', 'tier', 'location', 'promotion', 'customer_group')),
    CONSTRAINT product_prices_min_quantity_check
        CHECK (min_quantity > 0),
    CONSTRAINT product_prices_price_check
        CHECK (price >= 0),
    CONSTRAINT product_prices_date_window_check
        CHECK (starts_at IS NULL OR ends_at IS NULL OR starts_at <= ends_at)
);

CREATE TRIGGER set_product_prices_updated_at
BEFORE UPDATE ON product_prices
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_product_prices_business_product_active
    ON product_prices (business_id, product_id, active, priority);

CREATE INDEX IF NOT EXISTS idx_product_prices_type_active
    ON product_prices (business_id, price_type, active);

CREATE INDEX IF NOT EXISTS idx_product_prices_location_id
    ON product_prices (location_id)
    WHERE location_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_product_prices_customer_group
    ON product_prices (business_id, LOWER(customer_group))
    WHERE customer_group IS NOT NULL;
