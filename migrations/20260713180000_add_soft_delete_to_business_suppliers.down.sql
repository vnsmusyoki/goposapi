DROP INDEX IF EXISTS idx_business_suppliers_deleted_at;
DROP INDEX IF EXISTS idx_business_suppliers_business_contact_active;

ALTER TABLE business_suppliers
    DROP COLUMN IF EXISTS deleted_by,
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS deleted;

ALTER TABLE business_suppliers
    ADD CONSTRAINT business_suppliers_business_id_contact_id_key UNIQUE (business_id, contact_id);
