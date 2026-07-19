ALTER TABLE sales_orders
    ADD COLUMN IF NOT EXISTS requires_further_action BOOLEAN NOT NULL DEFAULT TRUE;

UPDATE sales_orders so
SET requires_further_action = s.requires_further_action
FROM sale_order_statuses s
WHERE s.code = so.status;
