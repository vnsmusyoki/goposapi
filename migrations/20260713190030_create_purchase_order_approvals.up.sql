CREATE TABLE IF NOT EXISTS purchase_order_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    approval_status VARCHAR(30) NOT NULL DEFAULT 'pending_approval',
    reminder_channels TEXT[] NOT NULL DEFAULT ARRAY['notification']::text[],
    reminder_message TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    requested_by UUID REFERENCES users(id) ON DELETE SET NULL,
    actioned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actioned_at TIMESTAMPTZ,
    reminder_sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT purchase_order_approvals_status_check
        CHECK (approval_status IN ('pending_approval', 'approved', 'rejected', 'cancelled')),
    CONSTRAINT purchase_order_approvals_channels_check
        CHECK (cardinality(reminder_channels) > 0)
);

CREATE TRIGGER set_purchase_order_approvals_updated_at
BEFORE UPDATE ON purchase_order_approvals
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_purchase_order_approvals_business_id
    ON purchase_order_approvals (business_id);

CREATE INDEX IF NOT EXISTS idx_purchase_order_approvals_purchase_order_id
    ON purchase_order_approvals (purchase_order_id);

CREATE INDEX IF NOT EXISTS idx_purchase_order_approvals_status
    ON purchase_order_approvals (approval_status);
