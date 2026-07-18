DROP INDEX IF EXISTS idx_sales_orders_customer_id;

ALTER TABLE sales_orders
    DROP COLUMN IF EXISTS customer_id;
