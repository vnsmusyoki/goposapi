ALTER TABLE purchase_orders
    ADD COLUMN IF NOT EXISTS delivery_address TEXT,
    ADD COLUMN IF NOT EXISTS delivery_charges NUMERIC(14,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS delivery_document_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS delivery_document_url TEXT,
    ADD COLUMN IF NOT EXISTS order_discount_amount NUMERIC(14,4) NOT NULL DEFAULT 0;

ALTER TABLE purchase_orders
    DROP CONSTRAINT IF EXISTS purchase_orders_numeric_non_negative;

ALTER TABLE purchase_orders
    ADD CONSTRAINT purchase_orders_numeric_non_negative
        CHECK (
            payment_term_value >= 0
            AND subtotal >= 0
            AND total_discount >= 0
            AND total_tax >= 0
            AND grand_total >= 0
            AND items_count >= 0
            AND total_quantity >= 0
            AND delivery_charges >= 0
            AND order_discount_amount >= 0
        );

CREATE TABLE IF NOT EXISTS purchase_order_additional_expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_order_id UUID REFERENCES purchase_orders(id) ON DELETE CASCADE,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    expense_name VARCHAR(255) NOT NULL,
    amount NUMERIC(14,4) NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT purchase_order_additional_expenses_amount_check CHECK (amount >= 0)
);

CREATE TRIGGER set_purchase_order_additional_expenses_updated_at
BEFORE UPDATE ON purchase_order_additional_expenses
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_purchase_order_additional_expenses_purchase_order_id
    ON purchase_order_additional_expenses (purchase_order_id);

CREATE INDEX IF NOT EXISTS idx_purchase_order_additional_expenses_business_id
    ON purchase_order_additional_expenses (business_id);

CREATE INDEX IF NOT EXISTS idx_purchase_order_additional_expenses_deleted_at
    ON purchase_order_additional_expenses (deleted_at);
