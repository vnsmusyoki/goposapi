CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    sku VARCHAR(255) NOT NULL,
    barcode VARCHAR(255) DEFAULT '',
    product_type VARCHAR(20) NOT NULL DEFAULT 'single',
    unit_id UUID NULL REFERENCES business_units(id) ON DELETE SET NULL,
    brand_id UUID NULL REFERENCES product_brands(id) ON DELETE SET NULL,
    category_id UUID NULL REFERENCES product_categories(id) ON DELETE SET NULL,
    sub_category_id INTEGER NULL REFERENCES product_sub_categories(id) ON DELETE SET NULL,
    is_for_selling BOOLEAN NOT NULL DEFAULT TRUE,
    manage_stock BOOLEAN NOT NULL DEFAULT FALSE,
    alert_quantity INTEGER NULL,
    tax_type VARCHAR(20) NOT NULL DEFAULT 'exclusive',
    tax_rate NUMERIC(10,2) NOT NULL DEFAULT 0,
    default_purchase_price NUMERIC(14,4) NULL,
    purchase_price_exclusive NUMERIC(14,4) NULL,
    purchase_price_inclusive NUMERIC(14,4) NULL,
    profit_margin NUMERIC(10,2) NULL,
    default_selling_price NUMERIC(14,4) NULL,
    description TEXT,
    brochure_name VARCHAR(255),
    brochure_url TEXT,
    currency_code VARCHAR(20) NOT NULL DEFAULT 'USD',
    currency_symbol_placement VARCHAR(10) NOT NULL DEFAULT 'before',
    currency_precision INTEGER NOT NULL DEFAULT 2,
    all_locations BOOLEAN NOT NULL DEFAULT FALSE,
    has_warranty BOOLEAN NOT NULL DEFAULT FALSE,
    warranty_duration VARCHAR(50),
    warranty_period VARCHAR(20),
    warranty_coverage TEXT,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT products_product_type_check
        CHECK (product_type IN ('single', 'combo', 'variable')),
    CONSTRAINT products_tax_type_check
        CHECK (tax_type IN ('exclusive', 'inclusive', 'none')),
    CONSTRAINT products_currency_symbol_placement_check
        CHECK (currency_symbol_placement IN ('before', 'after')),
    CONSTRAINT products_alert_quantity_check
        CHECK (alert_quantity IS NULL OR alert_quantity >= 2),
    CONSTRAINT products_numeric_pricing_non_negative
        CHECK (
            COALESCE(default_purchase_price, 0) >= 0
            AND COALESCE(purchase_price_exclusive, 0) >= 0
            AND COALESCE(purchase_price_inclusive, 0) >= 0
            AND COALESCE(profit_margin, 0) >= 0
            AND COALESCE(default_selling_price, 0) >= 0
            AND tax_rate >= 0
        )
);

CREATE TRIGGER set_products_updated_at
BEFORE UPDATE ON products
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_products_business_id
    ON products (business_id);

CREATE INDEX IF NOT EXISTS idx_products_business_created_at
    ON products (business_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_products_deleted_at
    ON products (deleted_at);

CREATE INDEX IF NOT EXISTS idx_products_business_product_type
    ON products (business_id, product_type);

CREATE UNIQUE INDEX IF NOT EXISTS idx_products_business_sku_active
    ON products (business_id, LOWER(sku))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_products_business_name_active
    ON products (business_id, LOWER(name))
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS product_sub_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    unit_id UUID NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_product_sub_units_updated_at
BEFORE UPDATE ON product_sub_units
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_sub_units_unique_active
    ON product_sub_units (product_id, unit_id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS product_locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES business_locations(id) ON DELETE CASCADE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_product_locations_updated_at
BEFORE UPDATE ON product_locations
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_product_locations_product_id
    ON product_locations (product_id);

CREATE INDEX IF NOT EXISTS idx_product_locations_location_id
    ON product_locations (location_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_locations_unique_active
    ON product_locations (product_id, location_id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS product_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    image_url TEXT NOT NULL,
    image_name VARCHAR(255),
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_product_images_updated_at
BEFORE UPDATE ON product_images
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_product_images_product_id
    ON product_images (product_id);

CREATE INDEX IF NOT EXISTS idx_product_images_deleted_at
    ON product_images (deleted_at);

CREATE TABLE IF NOT EXISTS product_combo_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    combo_product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    item_product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    item_name VARCHAR(255) NOT NULL,
    item_sku VARCHAR(255) NOT NULL,
    item_unit VARCHAR(100) NOT NULL DEFAULT '',
    quantity NUMERIC(14,4) NOT NULL DEFAULT 1,
    price_each NUMERIC(14,4) NOT NULL DEFAULT 0,
    subtotal NUMERIC(14,4) NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT product_combo_items_numeric_non_negative
        CHECK (quantity > 0 AND price_each >= 0 AND subtotal >= 0)
);

CREATE TRIGGER set_product_combo_items_updated_at
BEFORE UPDATE ON product_combo_items
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_product_combo_items_business_id
    ON product_combo_items (business_id);

CREATE INDEX IF NOT EXISTS idx_product_combo_items_combo_product_id
    ON product_combo_items (combo_product_id);

CREATE INDEX IF NOT EXISTS idx_product_combo_items_item_product_id
    ON product_combo_items (item_product_id);

CREATE INDEX IF NOT EXISTS idx_product_combo_items_deleted_at
    ON product_combo_items (deleted_at);

CREATE TABLE IF NOT EXISTS product_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    sku VARCHAR(255) NOT NULL,
    barcode VARCHAR(255) DEFAULT '',
    cost NUMERIC(14,4) NOT NULL DEFAULT 0,
    selling NUMERIC(14,4) NOT NULL DEFAULT 0,
    stock NUMERIC(14,4) NOT NULL DEFAULT 0,
    show_optional_fields BOOLEAN NOT NULL DEFAULT FALSE,
    weight VARCHAR(100),
    length VARCHAR(100),
    width VARCHAR(100),
    height VARCHAR(100),
    image_name VARCHAR(255),
    image_url TEXT,
    reorder_level INTEGER,
    expiry_date DATE,
    supplier_code VARCHAR(255),
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT product_variants_numeric_non_negative
        CHECK (cost >= 0 AND selling >= 0 AND stock >= 0 AND COALESCE(reorder_level, 0) >= 0)
);

CREATE TRIGGER set_product_variants_updated_at
BEFORE UPDATE ON product_variants
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_product_variants_business_id
    ON product_variants (business_id);

CREATE INDEX IF NOT EXISTS idx_product_variants_product_id
    ON product_variants (product_id);

CREATE INDEX IF NOT EXISTS idx_product_variants_deleted_at
    ON product_variants (deleted_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_variants_business_product_sku_active
    ON product_variants (business_id, product_id, LOWER(sku))
    WHERE deleted_at IS NULL;
