CREATE TABLE IF NOT EXISTS cash_register_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    cash_register_id UUID NOT NULL REFERENCES cash_registers(id) ON DELETE CASCADE,
    sale_id UUID NULL REFERENCES sales(id) ON DELETE SET NULL,
    transaction_type VARCHAR(40) NOT NULL,
    payment_method VARCHAR(50) NOT NULL DEFAULT 'cash',
    amount NUMERIC(14,4) NOT NULL,
    reference_number VARCHAR(100) NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT cash_register_transactions_type_check
        CHECK (transaction_type IN ('opening_cash', 'sale_payment', 'refund', 'cash_in', 'cash_out', 'adjustment')),
    CONSTRAINT cash_register_transactions_amount_non_negative
        CHECK (amount >= 0)
);

CREATE INDEX IF NOT EXISTS idx_cash_register_transactions_business_id
    ON cash_register_transactions (business_id);

CREATE INDEX IF NOT EXISTS idx_cash_register_transactions_register_id
    ON cash_register_transactions (cash_register_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_cash_register_transactions_sale_id
    ON cash_register_transactions (sale_id)
    WHERE sale_id IS NOT NULL;
