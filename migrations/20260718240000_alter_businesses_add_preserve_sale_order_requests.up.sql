ALTER TABLE businesses
    ADD COLUMN IF NOT EXISTS preserve_sale_order_requests BOOLEAN NOT NULL DEFAULT FALSE;
