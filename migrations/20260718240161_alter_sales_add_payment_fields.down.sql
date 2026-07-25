DROP INDEX IF EXISTS idx_sales_payment_status;
DROP INDEX IF EXISTS idx_sales_customer_id;

ALTER TABLE sales
    DROP COLUMN IF EXISTS payment_status,
    DROP COLUMN IF EXISTS balance_due,
    DROP COLUMN IF EXISTS credit_amount,
    DROP COLUMN IF EXISTS paid_amount,
    DROP COLUMN IF EXISTS customer_id;
