ALTER TABLE purchase_order_items
    DROP CONSTRAINT IF EXISTS purchase_order_items_numeric_non_negative;

ALTER TABLE purchase_order_items
    DROP COLUMN IF EXISTS selling_price;

ALTER TABLE purchase_order_items
    ADD CONSTRAINT purchase_order_items_numeric_non_negative
        CHECK (
            order_quantity > 0
            AND unit_cost_before_discount >= 0
            AND discount_percentage >= 0
            AND discount_amount >= 0
            AND unit_cost_before_tax >= 0
            AND product_tax_rate >= 0
            AND tax_amount >= 0
            AND net_cost >= 0
            AND line_cost >= 0
            AND COALESCE(received_quantity, 0) >= 0
            AND items_received >= 0
        );
