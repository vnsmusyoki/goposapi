DROP INDEX IF EXISTS idx_purchase_order_items_balance_quantity;
ALTER TABLE purchase_order_items
    DROP COLUMN IF EXISTS balance_quantity;
