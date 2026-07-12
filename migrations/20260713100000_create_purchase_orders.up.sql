CREATE TABLE IF NOT EXISTS purchase_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    supplier_id UUID NOT NULL REFERENCES business_suppliers(id) ON DELETE RESTRICT,
    location_id UUID NOT NULL REFERENCES business_locations(id) ON DELETE RESTRICT,
    reference_number VARCHAR(100) NOT NULL DEFAULT '',
    order_date DATE NOT NULL,
    delivery_date DATE,
    payment_term_value INTEGER NOT NULL DEFAULT 0,
    payment_term_unit VARCHAR(20) NOT NULL DEFAULT 'days',
    attachment_name VARCHAR(255),
    attachment_url TEXT,
    notes TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'draft',
    delivery_status VARCHAR(30) NOT NULL DEFAULT 'pending_delivery',
    payment_status VARCHAR(30) NOT NULL DEFAULT 'unpaid',
    subtotal NUMERIC(14,4) NOT NULL DEFAULT 0,
    total_discount NUMERIC(14,4) NOT NULL DEFAULT 0,
    total_tax NUMERIC(14,4) NOT NULL DEFAULT 0,
    grand_total NUMERIC(14,4) NOT NULL DEFAULT 0,
    items_count INTEGER NOT NULL DEFAULT 0,
    total_quantity NUMERIC(14,4) NOT NULL DEFAULT 0,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT purchase_orders_numeric_non_negative
        CHECK (
            payment_term_value >= 0
            AND subtotal >= 0
            AND total_discount >= 0
            AND total_tax >= 0
            AND grand_total >= 0
            AND items_count >= 0
            AND total_quantity >= 0
        )
);

CREATE TRIGGER set_purchase_orders_updated_at
BEFORE UPDATE ON purchase_orders
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_purchase_orders_business_id
    ON purchase_orders (business_id);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_business_created_at
    ON purchase_orders (business_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_deleted_at
    ON purchase_orders (deleted_at);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_supplier_id
    ON purchase_orders (supplier_id);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_location_id
    ON purchase_orders (location_id);

CREATE TABLE IF NOT EXISTS purchase_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_order_id UUID REFERENCES purchase_orders(id) ON DELETE CASCADE,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    product_name VARCHAR(255) NOT NULL,
    sku VARCHAR(255) NOT NULL DEFAULT '',
    unit VARCHAR(100) NOT NULL DEFAULT '',
    order_quantity NUMERIC(14,4) NOT NULL DEFAULT 1,
    unit_cost_before_discount NUMERIC(14,4) NOT NULL DEFAULT 0,
    discount_percentage NUMERIC(10,2) NOT NULL DEFAULT 0,
    discount_amount NUMERIC(14,4) NOT NULL DEFAULT 0,
    unit_cost_before_tax NUMERIC(14,4) NOT NULL DEFAULT 0,
    product_tax_rate NUMERIC(10,2) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(14,4) NOT NULL DEFAULT 0,
    net_cost NUMERIC(14,4) NOT NULL DEFAULT 0,
    selling_price NUMERIC(14,4) NOT NULL DEFAULT 0,
    line_cost NUMERIC(14,4) NOT NULL DEFAULT 0,
    expiry_date DATE,
    lot_number VARCHAR(255),
    received_quantity NUMERIC(14,4),
    items_received NUMERIC(14,4) NOT NULL DEFAULT 0,
    received_status VARCHAR(30) NOT NULL DEFAULT 'pending',
    sort_order INTEGER NOT NULL DEFAULT 0,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT purchase_order_items_numeric_non_negative
        CHECK (
            order_quantity > 0
            AND unit_cost_before_discount >= 0
            AND discount_percentage >= 0
            AND discount_amount >= 0
            AND unit_cost_before_tax >= 0
            AND product_tax_rate >= 0
            AND tax_amount >= 0
            AND net_cost >= 0
            AND selling_price >= 0
            AND line_cost >= 0
            AND COALESCE(received_quantity, 0) >= 0
            AND items_received >= 0
        )
);

CREATE TRIGGER set_purchase_order_items_updated_at
BEFORE UPDATE ON purchase_order_items
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_purchase_order_items_purchase_order_id
    ON purchase_order_items (purchase_order_id);

CREATE INDEX IF NOT EXISTS idx_purchase_order_items_business_id
    ON purchase_order_items (business_id);

CREATE INDEX IF NOT EXISTS idx_purchase_order_items_deleted_at
    ON purchase_order_items (deleted_at);
