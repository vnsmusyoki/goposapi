CREATE TABLE IF NOT EXISTS cash_registers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    business_location_id UUID NOT NULL REFERENCES business_locations(id) ON DELETE RESTRICT,
    register_number VARCHAR(100) NOT NULL DEFAULT '',
    status VARCHAR(30) NOT NULL DEFAULT 'open',
    opened_by UUID REFERENCES users(id) ON DELETE SET NULL,
    closed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    opened_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMPTZ NULL,
    opening_cash_amount NUMERIC(14,4) NOT NULL DEFAULT 0,
    cash_sales_amount NUMERIC(14,4) NOT NULL DEFAULT 0,
    cash_refund_amount NUMERIC(14,4) NOT NULL DEFAULT 0,
    cash_in_amount NUMERIC(14,4) NOT NULL DEFAULT 0,
    cash_out_amount NUMERIC(14,4) NOT NULL DEFAULT 0,
    expected_closing_cash_amount NUMERIC(14,4) NOT NULL DEFAULT 0,
    actual_closing_cash_amount NUMERIC(14,4) NULL,
    reconciled BOOLEAN NOT NULL DEFAULT FALSE,
    reconciliation_difference_amount NUMERIC(14,4) NULL,
    notes TEXT NOT NULL DEFAULT '',
    closing_note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT cash_registers_status_check
        CHECK (status IN ('open', 'closed', 'suspended')),
    CONSTRAINT cash_registers_amounts_non_negative
        CHECK (
            opening_cash_amount >= 0
            AND cash_sales_amount >= 0
            AND cash_refund_amount >= 0
            AND cash_in_amount >= 0
            AND cash_out_amount >= 0
            AND expected_closing_cash_amount >= 0
            AND (actual_closing_cash_amount IS NULL OR actual_closing_cash_amount >= 0)
        ),
    CONSTRAINT cash_registers_closed_state_check
        CHECK (
            (status <> 'closed' AND closed_at IS NULL)
            OR (status = 'closed' AND closed_at IS NOT NULL)
        )
);

CREATE TRIGGER set_cash_registers_updated_at
BEFORE UPDATE ON cash_registers
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_cash_registers_business_id
    ON cash_registers (business_id);

CREATE INDEX IF NOT EXISTS idx_cash_registers_business_location_status
    ON cash_registers (business_id, business_location_id, status);

CREATE INDEX IF NOT EXISTS idx_cash_registers_opened_at
    ON cash_registers (business_id, opened_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_cash_registers_one_open_per_location
    ON cash_registers (business_id, business_location_id)
    WHERE status = 'open';
