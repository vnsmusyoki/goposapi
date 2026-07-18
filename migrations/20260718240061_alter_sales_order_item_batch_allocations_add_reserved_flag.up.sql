ALTER TABLE sales_order_item_batch_allocations
    ADD COLUMN IF NOT EXISTS is_reserved BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_sales_order_item_batch_allocations_reserved
    ON sales_order_item_batch_allocations (is_reserved);
