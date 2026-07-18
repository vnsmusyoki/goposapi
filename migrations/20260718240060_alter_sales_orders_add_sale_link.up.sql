ALTER TABLE sales_orders
    ADD COLUMN IF NOT EXISTS sale_id UUID REFERENCES sales(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS converted_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_sales_orders_sale_id
    ON sales_orders (sale_id)
    WHERE sale_id IS NOT NULL;
