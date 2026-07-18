DROP TRIGGER IF EXISTS trg_set_product_profit_amount ON products;
DROP FUNCTION IF EXISTS set_product_profit_amount();

ALTER TABLE products
    DROP COLUMN IF EXISTS profit_amount;
