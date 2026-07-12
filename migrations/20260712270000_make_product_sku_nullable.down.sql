UPDATE products
SET sku = CONCAT('SKU-', id::text)
WHERE sku IS NULL;

ALTER TABLE products
    ALTER COLUMN sku SET NOT NULL;
