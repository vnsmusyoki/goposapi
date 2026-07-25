CREATE TABLE IF NOT EXISTS payment_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    alias VARCHAR(40) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    is_credit BOOLEAN NOT NULL DEFAULT FALSE,
    requires_reference BOOLEAN NOT NULL DEFAULT FALSE,
    requires_phone BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT payment_methods_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT payment_methods_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT payment_methods_business_code_unique UNIQUE (business_id, code)
);

CREATE TRIGGER set_payment_methods_updated_at
BEFORE UPDATE ON payment_methods
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_payment_methods_business_enabled
    ON payment_methods (business_id, is_enabled, sort_order);

INSERT INTO payment_methods (
    business_id,
    code,
    name,
    alias,
    description,
    is_enabled,
    is_credit,
    requires_reference,
    requires_phone,
    sort_order
)
SELECT
    b.id,
    method.code,
    method.name,
    method.alias,
    method.description,
    method.is_enabled,
    method.is_credit,
    method.requires_reference,
    method.requires_phone,
    method.sort_order
FROM businesses b
CROSS JOIN (
    VALUES
        ('cash', 'Cash Payment', 'Cash', 'Cash received at the register.', TRUE, FALSE, FALSE, FALSE, 10),
        ('cheque', 'Cheque Payment', 'Cheque', 'Cheque payment received from the customer.', TRUE, FALSE, TRUE, FALSE, 20),
        ('card', 'Card Payment', 'Card', 'Card or terminal payment.', TRUE, FALSE, TRUE, FALSE, 30),
        ('bank_transfer', 'Bank Transfer', 'Bank', 'Bank transfer payment.', TRUE, FALSE, TRUE, FALSE, 40),
        ('advance', 'Advance Payment', 'Advance', 'Payment made from customer advance balance.', TRUE, FALSE, TRUE, FALSE, 50),
        ('mpesa', 'MPESA', 'MPesa', 'MPesa STK push or till payment.', TRUE, FALSE, TRUE, TRUE, 60),
        ('other', 'Other Payments', 'Other', 'Any other enabled payment method.', TRUE, FALSE, TRUE, FALSE, 70),
        ('credit', 'Credit Sale', 'Credit', 'Customer credit sale.', TRUE, TRUE, FALSE, FALSE, 80)
) AS method(code, name, alias, description, is_enabled, is_credit, requires_reference, requires_phone, sort_order)
ON CONFLICT (business_id, code) DO NOTHING;
