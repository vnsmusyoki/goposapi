DROP INDEX IF EXISTS idx_sale_payments_method;
DROP INDEX IF EXISTS idx_sale_payments_sale_id;
DROP INDEX IF EXISTS idx_sale_payments_business_id;
DROP TABLE IF EXISTS sale_payments;
DROP INDEX IF EXISTS idx_payment_methods_business_enabled;
DROP TRIGGER IF EXISTS set_payment_methods_updated_at ON payment_methods;
DROP TABLE IF EXISTS payment_methods;
