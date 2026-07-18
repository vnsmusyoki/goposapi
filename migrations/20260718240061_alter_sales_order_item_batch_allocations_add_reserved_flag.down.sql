DROP INDEX IF EXISTS idx_sales_order_item_batch_allocations_reserved;

ALTER TABLE sales_order_item_batch_allocations
    DROP COLUMN IF EXISTS is_reserved;
