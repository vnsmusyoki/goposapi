CREATE TABLE IF NOT EXISTS subscription_payments(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    business_subscription_id UUID NOT NULL
        REFERENCES business_subscriptions(id)
        ON DELETE CASCADE,

    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    package_id UUID NOT NULL REFERENCES packages(id),

    amount NUMERIC(12, 2) NOT NULL CHECK (amount >= 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'KES',

    payment_method VARCHAR(50),
    payment_reference VARCHAR(255),

    status VARCHAR(30) NOT NULL DEFAULT 'pending',

    paid_at TIMESTAMPTZ,

    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
