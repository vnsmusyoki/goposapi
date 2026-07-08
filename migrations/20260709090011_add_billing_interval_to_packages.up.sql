ALTER TABLE packages
ADD COLUMN billing_interval_id UUID;

UPDATE packages
SET billing_interval_id = billing_intervals.id
FROM billing_intervals
WHERE billing_intervals.code = 'monthly'
  AND packages.billing_interval_id IS NULL;

ALTER TABLE packages
ALTER COLUMN billing_interval_id SET NOT NULL;

ALTER TABLE packages
ADD CONSTRAINT fk_packages_billing_interval
FOREIGN KEY (billing_interval_id)
REFERENCES billing_intervals(id)
ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_packages_billing_interval_id
    ON packages(billing_interval_id);
