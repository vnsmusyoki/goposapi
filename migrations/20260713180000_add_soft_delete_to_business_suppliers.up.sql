ALTER TABLE business_suppliers
    ADD COLUMN IF NOT EXISTS deleted BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS deleted_by UUID REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE business_suppliers
    DROP CONSTRAINT IF EXISTS business_suppliers_business_id_contact_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_business_suppliers_business_contact_active
    ON business_suppliers (business_id, contact_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_business_suppliers_deleted_at
    ON business_suppliers (deleted_at);
