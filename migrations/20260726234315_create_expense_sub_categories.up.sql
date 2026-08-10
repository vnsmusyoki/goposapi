CREATE TABLE IF NOT EXISTS expense_sub_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    expense_category_id UUID NOT NULL REFERENCES expense_categories(id) ON DELETE CASCADE,

    sub_category_code VARCHAR(50) NOT NULL,
    name VARCHAR(150) NOT NULL,
    description TEXT,

    color VARCHAR(20),
    icon VARCHAR(100),

    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (business_id, sub_category_code)
);

CREATE TRIGGER set_expense_sub_categories_updated_at
BEFORE UPDATE ON expense_sub_categories
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_expense_sub_categories_business_id
    ON expense_sub_categories (business_id);

CREATE INDEX IF NOT EXISTS idx_expense_sub_categories_expense_category_id
    ON expense_sub_categories (expense_category_id);

CREATE INDEX IF NOT EXISTS idx_expense_sub_categories_business_category_id
    ON expense_sub_categories (business_id, expense_category_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_expense_sub_categories_business_category_name
    ON expense_sub_categories (business_id, expense_category_id, LOWER(name));
