CREATE TABLE IF NOT EXISTS billing_intervals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,

    interval_months INTEGER,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_billing_intervals_updated_at
BEFORE UPDATE ON billing_intervals
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

INSERT INTO billing_intervals (code, name, description, interval_months, sort_order)
VALUES
    ('monthly', 'Monthly', 'Billed every month', 1, 1),
    ('quarterly', 'Quarterly', 'Billed every three months', 3, 2),
    ('yearly', 'Yearly', 'Billed every twelve months', 12, 3),
    ('lifetime', 'Lifetime', 'One-time lifetime access', NULL, 4)
ON CONFLICT (code) DO NOTHING;
