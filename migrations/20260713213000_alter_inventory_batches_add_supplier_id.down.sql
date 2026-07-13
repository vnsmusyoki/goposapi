DROP INDEX IF EXISTS idx_inventory_batches_supplier_id;

ALTER TABLE inventory_batches
DROP COLUMN IF EXISTS supplier_id;
