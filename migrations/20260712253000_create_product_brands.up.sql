CREATE TABLE IF NOT EXISTS product_brands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    short_description TEXT NOT NULL DEFAULT '',
    added_by UUID REFERENCES users(id) ON DELETE SET NULL,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_product_brands_updated_at
BEFORE UPDATE ON product_brands
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_product_brands_business_id
    ON product_brands(business_id);

CREATE INDEX IF NOT EXISTS idx_product_brands_deleted_at
    ON product_brands(deleted_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_brands_business_name_active
    ON product_brands(business_id, LOWER(name))
    WHERE deleted_at IS NULL;
