CREATE TABLE IF NOT EXISTS sales_order_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    sales_order_id UUID NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,
    action VARCHAR(100) NOT NULL,
    actioned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    note TEXT NOT NULL DEFAULT '',
    action_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sales_order_logs_business_id
    ON sales_order_logs (business_id);

CREATE INDEX IF NOT EXISTS idx_sales_order_logs_sales_order_id
    ON sales_order_logs (sales_order_id);

CREATE INDEX IF NOT EXISTS idx_sales_order_logs_action_date
    ON sales_order_logs (action_date DESC);
