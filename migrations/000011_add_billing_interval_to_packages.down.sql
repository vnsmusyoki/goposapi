DROP INDEX IF EXISTS idx_packages_billing_interval_id;

ALTER TABLE packages
DROP CONSTRAINT IF EXISTS fk_packages_billing_interval;

ALTER TABLE packages
DROP COLUMN IF EXISTS billing_interval_id;
