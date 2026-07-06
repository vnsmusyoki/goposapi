DROP TRIGGER IF EXISTS set_businesses_updated_at ON businesses;
DROP TABLE IF EXISTS businesses;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP EXTENSION IF EXISTS pgcrypto;
