ALTER TABLE sales_orders
    DROP CONSTRAINT IF EXISTS fk_sales_orders_status;

DROP INDEX IF EXISTS idx_sales_orders_status;

ALTER TABLE sales_orders
    DROP COLUMN IF EXISTS requires_further_action;

DROP TRIGGER IF EXISTS set_sale_order_statuses_updated_at ON sale_order_statuses;

DROP INDEX IF EXISTS idx_sale_order_statuses_sort_order;

DROP TABLE IF EXISTS sale_order_statuses;
