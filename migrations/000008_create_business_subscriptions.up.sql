CREATE TABLE IF NOT EXISTS business_subscriptions(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    package_id UUID NOT NULL REFERENCES packages(id),

    status VARCHAR(30) NOT NULL DEFAULT 'trialing',

    starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_period_end TIMESTAMPTZ NOT NULL,

    trial_ends_at TIMESTAMPTZ NULL,

    cancelled_at TIMESTAMPTZ NULL,

    auto_renew BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);