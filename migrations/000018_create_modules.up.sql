CREATE TABLE IF NOT EXISTS modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    icon VARCHAR(100),
    path VARCHAR(255),
    workspace_type VARCHAR(50) NOT NULL DEFAULT 'admin',
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_modules_updated_at
BEFORE UPDATE ON modules
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS submodules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id UUID NOT NULL
        REFERENCES modules(id)
        ON DELETE CASCADE,
    code VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    icon VARCHAR(100),
    path VARCHAR(255),
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_submodules_updated_at
BEFORE UPDATE ON submodules
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS user_modules (
    user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    module_id UUID NOT NULL
        REFERENCES modules(id)
        ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, module_id)
);

CREATE INDEX IF NOT EXISTS idx_modules_workspace_type_active_sort
    ON modules(workspace_type, is_active, sort_order);

CREATE INDEX IF NOT EXISTS idx_submodules_module_id_active_sort
    ON submodules(module_id, is_active, sort_order);

CREATE INDEX IF NOT EXISTS idx_user_modules_module_id
    ON user_modules(module_id);

INSERT INTO modules (code, name, description, icon, path, workspace_type, sort_order)
VALUES
    ('admin_dashboard', 'Dashboard', 'Admin workspace overview', 'LayoutDashboard', '/admin/dashboard', 'admin', 1),
    ('admin_sales', 'Sales', 'Admin sales management', 'ShoppingBag', '/admin/sales', 'admin', 2),
    ('admin_products', 'Products', 'Admin product management', 'Package', '/admin/products', 'admin', 3),
    ('admin_customers', 'Customers', 'Admin customer management', 'Users', '/admin/customers', 'admin', 4),
    ('admin_pos', 'Point of Sale', 'Admin point of sale access', 'CreditCard', '/admin/terminal', 'admin', 5),
    ('admin_reports', 'Reports', 'Admin reports and analytics', 'PieChart', '/admin/reports', 'admin', 6),
    ('admin_settings', 'Settings', 'Admin configuration area', 'Settings', '/admin/settings', 'admin', 7),
    ('business_dashboard', 'Dashboard', 'Business workspace overview', 'LayoutDashboard', '/business/dashboard', 'business', 1),
    ('business_sales', 'Sales', 'Business sales management', 'ShoppingBag', '/business/sales', 'business', 2),
    ('business_products', 'Products', 'Business product management', 'Package', '/business/products', 'business', 3),
    ('business_customers', 'Customers', 'Business customer management', 'Users', '/business/customers', 'business', 4),
    ('business_reports', 'Reports', 'Business reports and analytics', 'PieChart', '/business/reports', 'business', 5),
    ('business_settings', 'Settings', 'Business configuration area', 'Settings', '/business/settings', 'business', 6)
ON CONFLICT (code) DO NOTHING;

INSERT INTO submodules (module_id, code, name, description, icon, path, sort_order)
SELECT m.id, v.code, v.name, v.description, v.icon, v.path, v.sort_order
FROM modules m
JOIN (
    VALUES
        ('admin_sales', 'admin_sales_orders', 'Orders', 'Manage sales orders', 'ShoppingCart', '/admin/sales/orders', 1),
        ('admin_sales', 'admin_sales_returns', 'Returns', 'Manage returned orders', 'Undo2', '/admin/sales/returns', 2),
        ('admin_sales', 'admin_sales_invoices', 'Invoices', 'Review invoices', 'FileText', '/admin/sales/invoices', 3),
        ('admin_products', 'admin_products_catalog', 'Catalog', 'Browse product catalog', 'Box', '/admin/products/catalog', 1),
        ('admin_products', 'admin_products_categories', 'Categories', 'Manage product categories', 'Layers', '/admin/products/categories', 2),
        ('admin_products', 'admin_products_inventory', 'Inventory', 'Track stock levels', 'ClipboardList', '/admin/products/inventory', 3),
        ('admin_customers', 'admin_customers_all', 'All Customers', 'View all customers', 'User', '/admin/customers', 1),
        ('admin_customers', 'admin_customers_loyalty', 'Loyalty Program', 'Manage loyalty rewards', 'Award', '/admin/customers/loyalty', 2),
        ('admin_reports', 'admin_reports_analytics', 'Analytics', 'Sales analytics dashboard', 'TrendingUp', '/admin/reports/analytics', 1),
        ('admin_reports', 'admin_reports_tax', 'Tax Reports', 'Review tax summaries', 'Percent', '/admin/reports/tax', 2),
        ('admin_settings', 'admin_settings_store', 'Store Settings', 'Edit store details', 'Store', '/admin/settings/store', 1),
        ('admin_settings', 'admin_settings_payment', 'Payment Methods', 'Configure payment methods', 'CreditCard', '/admin/settings/payment', 2),
        ('admin_settings', 'admin_settings_shipping', 'Shipping', 'Configure shipping rules', 'Truck', '/admin/settings/shipping', 3),
        ('business_sales', 'business_sales_orders', 'Orders', 'Manage sales orders', 'ShoppingCart', '/business/sales/orders', 1),
        ('business_sales', 'business_sales_invoices', 'Invoices', 'Review invoices', 'FileText', '/business/sales/invoices', 2),
        ('business_products', 'business_products_catalog', 'Catalog', 'Browse product catalog', 'Box', '/business/products/catalog', 1),
        ('business_products', 'business_products_inventory', 'Inventory', 'Track stock levels', 'ClipboardList', '/business/products/inventory', 2),
        ('business_customers', 'business_customers_all', 'All Customers', 'View all customers', 'User', '/business/customers', 1),
        ('business_customers', 'business_customers_loyalty', 'Loyalty Program', 'Manage loyalty rewards', 'Award', '/business/customers/loyalty', 2),
        ('business_reports', 'business_reports_analytics', 'Analytics', 'Sales analytics dashboard', 'TrendingUp', '/business/reports/analytics', 1),
        ('business_reports', 'business_reports_tax', 'Tax Reports', 'Review tax summaries', 'Percent', '/business/reports/tax', 2),
        ('business_settings', 'business_settings_store', 'Store Settings', 'Edit store details', 'Store', '/business/settings/store', 1),
        ('business_settings', 'business_settings_payment', 'Payment Methods', 'Configure payment methods', 'CreditCard', '/business/settings/payment', 2)
) AS v(parent_code, code, name, description, icon, path, sort_order)
    ON v.parent_code = m.code
ON CONFLICT (code) DO NOTHING;

INSERT INTO user_modules (user_id, module_id)
SELECT u.id, m.id
FROM users u
JOIN modules m ON m.workspace_type = 'admin'
WHERE u.email = 'admin@flowpos.local'
ON CONFLICT DO NOTHING;
