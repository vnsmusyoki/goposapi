ALTER TABLE sales
    ADD COLUMN IF NOT EXISTS customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS paid_amount NUMERIC(14,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS credit_amount NUMERIC(14,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS balance_due NUMERIC(14,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS payment_status VARCHAR(30) NOT NULL DEFAULT 'unpaid';

CREATE INDEX IF NOT EXISTS idx_sales_customer_id
    ON sales (customer_id);

CREATE INDEX IF NOT EXISTS idx_sales_payment_status
    ON sales (business_id, payment_status);
