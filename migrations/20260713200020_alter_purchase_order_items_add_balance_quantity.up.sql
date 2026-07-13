ALTER TABLE purchase_order_items
    ADD COLUMN IF NOT EXISTS balance_quantity NUMERIC(14,4);

UPDATE purchase_order_items
SET balance_quantity = COALESCE(order_quantity, 0)
WHERE balance_quantity IS NULL;

CREATE INDEX IF NOT EXISTS idx_purchase_order_items_balance_quantity
    ON purchase_order_items (balance_quantity);
