DROP INDEX IF EXISTS idx_payment_methods_business_enabled;
DROP TRIGGER IF EXISTS set_payment_methods_updated_at ON payment_methods;
DROP TABLE IF EXISTS payment_methods;
