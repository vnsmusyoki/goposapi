CREATE TABLE IF NOT EXISTS purchase_returns_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    purchase_return_id UUID NOT NULL REFERENCES purchase_returns(id) ON DELETE CASCADE,
    action VARCHAR(500) NOT NULL,
    actioned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    note TEXT NOT NULL DEFAULT '',
    action_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_purchase_returns_logs_business_id
    ON purchase_returns_logs (business_id);

CREATE INDEX IF NOT EXISTS idx_purchase_returns_logs_purchase_return_id_action_date
    ON purchase_returns_logs (purchase_return_id, action_date DESC);

CREATE INDEX IF NOT EXISTS idx_purchase_returns_logs_action_date
    ON purchase_returns_logs (action_date DESC);
