CREATE TABLE IF NOT EXISTS business_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    short_name VARCHAR(50) NOT NULL,
    allow_decimal BOOLEAN NOT NULL DEFAULT FALSE,
    is_multiple_of_other BOOLEAN NOT NULL DEFAULT FALSE,
    base_unit_id UUID NULL,
    conversion_rate NUMERIC(14,4) NOT NULL DEFAULT 0,
    created_by_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    created_by VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_business_units_base_unit
        FOREIGN KEY (base_unit_id)
        REFERENCES business_units(id)
        ON DELETE SET NULL,
    CONSTRAINT business_units_base_unit_consistency
        CHECK (
            (is_multiple_of_other = FALSE AND base_unit_id IS NULL AND conversion_rate = 0)
            OR
            (is_multiple_of_other = TRUE AND base_unit_id IS NOT NULL AND conversion_rate > 0)
        ),
    CONSTRAINT business_units_conversion_rate_non_negative
        CHECK (conversion_rate >= 0)
);

CREATE TRIGGER set_business_units_updated_at
BEFORE UPDATE ON business_units
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_business_units_business_id
    ON business_units (business_id);

CREATE INDEX IF NOT EXISTS idx_business_units_business_created_at
    ON business_units (business_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_business_units_base_unit_id
    ON business_units (business_id, base_unit_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_business_units_business_name
    ON business_units (business_id, LOWER(name));

CREATE UNIQUE INDEX IF NOT EXISTS idx_business_units_business_short_name
    ON business_units (business_id, LOWER(short_name));
