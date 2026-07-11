CREATE TABLE IF NOT EXISTS product_sub_categories (
    id SERIAL PRIMARY KEY,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    parent_category_id UUID NOT NULL REFERENCES product_categories(id) ON DELETE CASCADE,
    sub_category_code VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    meta_title VARCHAR(255),
    meta_description TEXT,
    image_url TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    featured BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_product_sub_categories_updated_at
BEFORE UPDATE ON product_sub_categories
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_product_sub_categories_business_id
    ON product_sub_categories(business_id);

CREATE INDEX IF NOT EXISTS idx_product_sub_categories_parent_category_id
    ON product_sub_categories(parent_category_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_sub_categories_business_code
    ON product_sub_categories(business_id, sub_category_code);

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_sub_categories_business_parent_name
    ON product_sub_categories(business_id, parent_category_id, LOWER(name));
