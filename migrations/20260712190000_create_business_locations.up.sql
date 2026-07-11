CREATE TABLE IF NOT EXISTS business_locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    location_id VARCHAR(50) NOT NULL,
    location_name VARCHAR(150) NOT NULL,
    landmark VARCHAR(255),
    exact_address TEXT,
    city VARCHAR(100),
    zip_code VARCHAR(20),
    state VARCHAR(100),
    country VARCHAR(100) NOT NULL DEFAULT 'Kenya',
    latitude NUMERIC(10,7),
    longitude NUMERIC(10,7),
    mobile VARCHAR(30) NOT NULL,
    alternate_contact_number VARCHAR(30),
    email VARCHAR(255),
    website VARCHAR(255),
    invoice_scheme VARCHAR(50) NOT NULL DEFAULT 'default',
    pos_invoice_layout VARCHAR(50) NOT NULL DEFAULT 'default',
    sale_invoice_layout VARCHAR(50) NOT NULL DEFAULT 'default',
    default_selling_price_group VARCHAR(50) NOT NULL DEFAULT 'retail',
    payment_methods JSONB NOT NULL DEFAULT '["cash"]'::jsonb,
    kra_pin VARCHAR(50) NOT NULL,
    tax_jurisdiction VARCHAR(100) NOT NULL DEFAULT 'Kenya',
    is_vat_registered BOOLEAN NOT NULL DEFAULT FALSE,
    vat_number VARCHAR(50),
    default_tax_type VARCHAR(50),
    prices_include_tax BOOLEAN NOT NULL DEFAULT TRUE,
    issue_tax_invoices BOOLEAN NOT NULL DEFAULT TRUE,
    tax_note TEXT,
    etims_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    environment VARCHAR(20) NOT NULL DEFAULT 'sandbox',
    integration_type VARCHAR(20) NOT NULL DEFAULT 'OSCU',
    is_head_office_branch BOOLEAN NOT NULL DEFAULT FALSE,
    kra_branch_id VARCHAR(100),
    device_serial_number VARCHAR(100),
    cmc_key VARCHAR(255),
    auto_submit_invoices BOOLEAN NOT NULL DEFAULT TRUE,
    allow_offline_sales BOOLEAN NOT NULL DEFAULT TRUE,
    retry_failed_invoices BOOLEAN NOT NULL DEFAULT TRUE,
    print_qr_code BOOLEAN NOT NULL DEFAULT TRUE,
    print_fiscal_details BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_business_locations_updated_at
BEFORE UPDATE ON business_locations
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_business_locations_business_id
    ON business_locations(business_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_business_locations_business_location_id
    ON business_locations(business_id, location_id);
