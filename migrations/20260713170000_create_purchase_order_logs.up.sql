CREATE TABLE IF NOT EXISTS purchase_order_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    action VARCHAR(500) NOT NULL,
    actioned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    note TEXT NOT NULL DEFAULT '',
    action_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_purchase_order_logs_business_id
    ON purchase_order_logs (business_id);

CREATE INDEX IF NOT EXISTS idx_purchase_order_logs_purchase_order_id_action_date
    ON purchase_order_logs (purchase_order_id, action_date DESC);

CREATE INDEX IF NOT EXISTS idx_purchase_order_logs_action_date
    ON purchase_order_logs (action_date DESC);
