DROP TABLE IF EXISTS purchase_order_additional_expenses;

ALTER TABLE purchase_orders
    DROP CONSTRAINT IF EXISTS purchase_orders_numeric_non_negative;

ALTER TABLE purchase_orders
    DROP COLUMN IF EXISTS delivery_address,
    DROP COLUMN IF EXISTS delivery_charges,
    DROP COLUMN IF EXISTS delivery_document_name,
    DROP COLUMN IF EXISTS delivery_document_url,
    DROP COLUMN IF EXISTS order_discount_amount;

ALTER TABLE purchase_orders
    ADD CONSTRAINT purchase_orders_numeric_non_negative
        CHECK (
            payment_term_value >= 0
            AND subtotal >= 0
            AND total_discount >= 0
            AND total_tax >= 0
            AND grand_total >= 0
            AND items_count >= 0
            AND total_quantity >= 0
        );
