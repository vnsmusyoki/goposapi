ALTER TABLE sales_orders
    ADD COLUMN IF NOT EXISTS customer_id UUID REFERENCES customers(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_sales_orders_customer_id
    ON sales_orders (customer_id);
