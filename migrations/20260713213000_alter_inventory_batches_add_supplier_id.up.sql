ALTER TABLE inventory_batches
ADD COLUMN IF NOT EXISTS supplier_id UUID REFERENCES business_suppliers(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_inventory_batches_supplier_id
    ON inventory_batches (supplier_id);

UPDATE inventory_batches ib
SET supplier_id = po.supplier_id
FROM purchase_orders po
WHERE ib.source_type = 'purchase_order'
  AND ib.source_id = po.id
  AND ib.supplier_id IS NULL
  AND po.supplier_id IS NOT NULL;
