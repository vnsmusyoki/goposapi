ALTER TABLE inventory_batches
ADD COLUMN IF NOT EXISTS manufacture_date DATE;
