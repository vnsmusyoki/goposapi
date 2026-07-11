CREATE UNIQUE INDEX IF NOT EXISTS idx_business_locations_business_location_name
    ON business_locations (business_id, LOWER(location_name));
