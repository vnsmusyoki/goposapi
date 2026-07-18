ALTER TABLE inventory_balances
    ADD COLUMN IF NOT EXISTS quantity_at_hand NUMERIC(14,4)
    GENERATED ALWAYS AS (COALESCE(quantity_available, 0) + COALESCE(quantity_reserved, 0)) STORED;
