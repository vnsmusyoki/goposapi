CREATE TABLE IF NOT EXISTS packages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,

    price NUMERIC(12, 2) NOT NULL
        CHECK (price >= 0),

    currency VARCHAR(10) NOT NULL DEFAULT 'KES',

    trial_days INTEGER NOT NULL DEFAULT 0
        CHECK (trial_days >= 0),

    max_users INTEGER,
    max_branches INTEGER,
    max_products INTEGER,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_featured BOOLEAN NOT NULL DEFAULT FALSE,

    sort_order INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);