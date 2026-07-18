DROP INDEX IF EXISTS idx_sales_orders_sale_id;

ALTER TABLE sales_orders
    DROP COLUMN IF EXISTS converted_at,
    DROP COLUMN IF EXISTS sale_id;
