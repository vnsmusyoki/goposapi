CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    purchase_order_id UUID REFERENCES purchase_orders(id) ON DELETE CASCADE,
    purchase_order_status_code VARCHAR(30) REFERENCES purchase_order_statuses(code) ON DELETE SET NULL,
    channels TEXT[] NOT NULL DEFAULT ARRAY[]::text[],
    receivers TEXT[] NOT NULL DEFAULT ARRAY[]::text[],
    message TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_notifications_updated_at
BEFORE UPDATE ON notifications
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_notifications_business_id
    ON notifications (business_id);

CREATE INDEX IF NOT EXISTS idx_notifications_purchase_order_id
    ON notifications (purchase_order_id);

CREATE INDEX IF NOT EXISTS idx_notifications_status_code
    ON notifications (purchase_order_status_code);
