CREATE TABLE IF NOT EXISTS product_warranties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    duration_value INTEGER NOT NULL DEFAULT 0,
    duration_unit VARCHAR(10) NOT NULL DEFAULT 'days',
    added_by UUID REFERENCES users(id) ON DELETE SET NULL,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_product_warranties_duration_unit CHECK (duration_unit IN ('days', 'months')),
    CONSTRAINT chk_product_warranties_duration_value CHECK (duration_value >= 0)
);

CREATE TRIGGER set_product_warranties_updated_at
BEFORE UPDATE ON product_warranties
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_product_warranties_business_id
    ON product_warranties(business_id);

CREATE INDEX IF NOT EXISTS idx_product_warranties_deleted_at
    ON product_warranties(deleted_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_warranties_business_name_active
    ON product_warranties(business_id, LOWER(name))
    WHERE deleted_at IS NULL;
