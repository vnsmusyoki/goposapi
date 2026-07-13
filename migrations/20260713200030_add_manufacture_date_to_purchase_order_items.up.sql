ALTER TABLE purchase_order_items
    ADD COLUMN IF NOT EXISTS manufacture_date DATE;
