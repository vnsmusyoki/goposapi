CREATE TABLE IF NOT EXISTS expense_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,

    name VARCHAR(150) NOT NULL,
    slug VARCHAR(150) NOT NULL,
    code VARCHAR(30),

    description TEXT,
    parent_id UUID REFERENCES expense_categories(id) ON DELETE CASCADE,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INT NOT NULL DEFAULT 0,

    color VARCHAR(20),
    icon VARCHAR(100),

    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (business_id, slug),
    UNIQUE (business_id, code)
);

CREATE TRIGGER set_expense_categories_updated_at
BEFORE UPDATE ON expense_categories
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_expense_categories_business_id
    ON expense_categories (business_id);

CREATE INDEX IF NOT EXISTS idx_expense_categories_parent_id
    ON expense_categories (parent_id);

CREATE INDEX IF NOT EXISTS idx_expense_categories_business_parent_id
    ON expense_categories (business_id, parent_id);
