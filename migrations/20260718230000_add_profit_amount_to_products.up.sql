ALTER TABLE products
    ADD COLUMN IF NOT EXISTS profit_amount NUMERIC(14,4) NULL;

UPDATE products
SET profit_amount = CASE
    WHEN product_type = 'single' THEN COALESCE(purchase_price_exclusive, 0) - COALESCE(default_purchase_price, 0)
    ELSE NULL
END;

CREATE OR REPLACE FUNCTION set_product_profit_amount()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.product_type = 'single' THEN
        NEW.profit_amount := COALESCE(NEW.purchase_price_exclusive, 0) - COALESCE(NEW.default_purchase_price, 0);
    ELSE
        NEW.profit_amount := NULL;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_set_product_profit_amount ON products;
CREATE TRIGGER trg_set_product_profit_amount
BEFORE INSERT OR UPDATE OF product_type, default_purchase_price, purchase_price_exclusive
ON products
FOR EACH ROW
EXECUTE FUNCTION set_product_profit_amount();
