-- ==========================================
-- SEED SCRIPT: modules, sub_modules, user_modules
-- ==========================================

DO $$
DECLARE
    admin_role_id UUID;
    business_role_id UUID;
    cashier_role_id UUID;
    seeded_module_id UUID;
    i INT;

    -- Module arrays
    -- [category_name, code, name, description, icon, path, sort_order]
    admin_modules TEXT[][] := ARRAY[
        ['System Management', 'admin_dashboard', 'Dashboard', 'Platform overview and analytics', 'LayoutDashboard', '/dashboard', '1'],
        ['System Management', 'admin_system_settings', 'System Settings', 'Platform configuration and settings', 'Settings', '/settings', '2'],
        ['System Management', 'admin_audit_logs', 'Audit Logs', 'View platform audit logs', 'ClipboardList', '/security/audit-logs', '3'],
        ['System Management', 'admin_notifications', 'Notifications', 'Configure platform notifications', 'Bell', '/notifications', '4'],
        ['System Management', 'admin_feature_flags', 'Feature Flags', 'Manage feature toggles', 'ToggleLeft', '/feature-flags', '5'],
        ['System Management', 'admin_api_keys', 'API Keys', 'Manage API keys and integrations', 'KeyRound', '/api-keys', '6'],

        ['Business Management', 'admin_businesses', 'Businesses', 'Manage all businesses on the platform', 'Store', '/businesses', '1'],
        ['Business Management', 'admin_business_onboarding', 'Onboarding', 'Approve and onboard new businesses', 'CheckCircle', '/businesses/onboarding', '2'],
        ['Business Management', 'admin_business_subscriptions', 'Subscriptions', 'Manage business subscriptions', 'CreditCard', '/businesses/subscriptions', '3'],
        ['Business Management', 'admin_business_profiles', 'Business Profiles', 'View and edit business profiles', 'Building2', '/businesses/profiles', '4'],
        ['Business Management', 'admin_business_status', 'Business Status', 'Monitor business status and health', 'ActivitySquare', '/businesses/status', '5'],

        ['User Management', 'admin_users', 'Users', 'Manage platform admin users', 'Users', '/users', '1'],
        ['User Management', 'admin_roles', 'Roles', 'Create and manage user roles', 'Shield', '/users/roles', '2'],
        ['User Management', 'admin_permissions', 'Permissions', 'Configure role-based permissions', 'Lock', '/users/permissions', '3'],
        ['User Management', 'admin_staff', 'Staff Accounts', 'Manage staff accounts', 'UserCheck', '/users/staff', '4'],
        ['User Management', 'admin_sessions', 'Login Sessions', 'View active login sessions', 'MonitorDot', '/users/sessions', '5'],

        ['Billing & Payments', 'admin_invoices', 'Invoices', 'Manage platform invoices', 'FileText', '/billing/invoices', '1'],
        ['Billing & Payments', 'admin_payments', 'Payments', 'Manage platform payments', 'DollarSign', '/billing/payments', '2'],
        ['Billing & Payments', 'admin_revenue', 'Revenue Reports', 'Platform revenue reports', 'TrendingUp', '/billing/revenue', '3'],
        ['Billing & Payments', 'admin_transactions', 'Transactions', 'View all transactions', 'ArrowLeftRight', '/billing/transactions', '4'],

        ['Support', 'admin_tickets', 'Support Tickets', 'Handle support tickets', 'Ticket', '/support/tickets', '1'],
        ['Support', 'admin_complaints', 'Complaints', 'Manage user complaints', 'AlertTriangle', '/support/complaints', '2'],
        ['Support', 'admin_escalations', 'Escalations', 'Handle escalated issues', 'ArrowUpCircle', '/support/escalations', '3'],
        ['Support', 'admin_knowledge_base', 'Knowledge Base', 'Manage knowledge base articles', 'BookOpen', '/support/knowledge-base', '4'],

        ['Analytics & Reports', 'admin_system_analytics', 'System Analytics', 'Platform-wide system analytics', 'LineChart', '/analytics/system', '1'],
        ['Analytics & Reports', 'admin_business_performance', 'Business Performance', 'Monitor business performance', 'Zap', '/analytics/business-performance', '2'],
        ['Analytics & Reports', 'admin_active_users', 'Active Users', 'Track active users', 'Users2', '/analytics/active-users', '3'],
        ['Analytics & Reports', 'admin_usage_reports', 'Usage Reports', 'Platform usage reports', 'PieChart', '/analytics/usage', '4'],
        ['Analytics & Reports', 'admin_traffic_reports', 'Traffic Reports', 'Traffic analytics', 'TrendingUp', '/analytics/traffic', '5'],

        ['Security', 'admin_login_audit', 'Login Audit', 'View login audit logs', 'ShieldCheck', '/security/login-audit', '1'],
        ['Security', 'admin_failed_logins', 'Failed Logins', 'Monitor failed login attempts', 'ShieldAlert', '/security/failed-logins', '2'],
        ['Security', 'admin_blocked_users', 'Blocked Users', 'Manage blocked users', 'UserX', '/security/blocked-users', '3'],
        ['Security', 'admin_security_policies', 'Security Policies', 'Configure security policies', 'FileShield', '/security/policies', '4'],
        ['Security', 'admin_mfa', 'MFA Settings', 'Configure multi-factor authentication', 'Smartphone', '/security/mfa', '5']
    ];

    business_modules TEXT[][] := ARRAY[
        -- [category_name, code, name, description, icon, path, sort_order]
        ['Sales & Operations', 'business_dashboard', 'Dashboard', 'Business sales summary and daily overview', 'LayoutDashboard', '/dashboard', '1'],
        ['Sales & Operations', 'business_pos', 'Point of Sale', 'Process sales and checkout', 'ShoppingCart', '/pos', '2'],
        ['Sales & Operations', 'business_sales_history', 'Sales History', 'View sales history and transactions', 'History', '/sales/history', '3'],
        ['Sales & Operations', 'business_returns', 'Returns & Refunds', 'Process returns and refunds', 'RotateCcw', '/sales/returns', '4'],
        ['Sales & Operations', 'business_orders', 'Orders', 'Manage customer orders', 'Package', '/orders', '5'],

        ['Inventory Management', 'business_products', 'Products', 'Manage product catalog', 'Box', '/inventory/products', '1'],
        ['Inventory Management', 'business_categories', 'Categories', 'Manage product categories', 'FolderTree', '/inventory/categories', '2'],
        ['Inventory Management', 'business_stock', 'Stock Management', 'Manage stock levels and adjustments', 'RefreshCw', '/inventory/stock', '3'],
        ['Inventory Management', 'business_suppliers', 'Suppliers', 'Manage suppliers and purchase orders', 'Truck', '/inventory/suppliers', '4'],
        ['Inventory Management', 'business_purchase_orders', 'Purchase Orders', 'Create and manage purchase orders', 'FileText', '/inventory/purchase-orders', '5'],
        ['Inventory Management', 'business_inventory_reports', 'Inventory Reports', 'View inventory reports', 'PieChart', '/inventory/reports', '6'],

        ['Customer Management', 'business_customers', 'Customers', 'Manage customer profiles', 'Users', '/customers', '1'],
        ['Customer Management', 'business_loyalty', 'Loyalty Program', 'Manage customer loyalty programs', 'Gift', '/customers/loyalty', '2'],
        ['Customer Management', 'business_customer_segments', 'Customer Segments', 'View customer segmentation', 'Users2', '/customers/segments', '3'],
        ['Customer Management', 'business_customer_feedback', 'Customer Feedback', 'Manage customer feedback', 'MessageSquare', '/customers/feedback', '4'],

        ['Staff Management', 'business_staff', 'Staff List', 'Manage staff members', 'Users', '/staff', '1'],
        ['Staff Management', 'business_shifts', 'Shifts & Attendance', 'Manage staff shifts and attendance', 'Calendar', '/staff/shifts', '2'],
        ['Staff Management', 'business_staff_permissions', 'Staff Permissions', 'Set staff roles and permissions', 'Lock', '/staff/permissions', '3'],
        ['Staff Management', 'business_staff_performance', 'Staff Performance', 'Monitor staff performance', 'Trophy', '/staff/performance', '4'],
        ['Staff Management', 'business_payroll', 'Payroll', 'Manage staff payroll', 'DollarSign', '/staff/payroll', '5'],

        ['Financials', 'business_payments', 'Payments', 'Configure payment methods', 'CreditCard', '/financials/payments', '1'],
        ['Financials', 'business_transactions', 'Transactions', 'View all transactions', 'Receipt', '/financials/transactions', '2'],
        ['Financials', 'business_sales_reports', 'Sales Reports', 'View sales reports and analytics', 'TrendingUp', '/financials/sales-reports', '3'],
        ['Financials', 'business_profit_loss', 'Profit & Loss', 'View profit and loss reports', 'DollarSign', '/financials/profit-loss', '4'],
        ['Financials', 'business_tax', 'Tax Management', 'Manage tax settings and reports', 'ReceiptText', '/financials/tax', '5'],
        ['Financials', 'business_expenses', 'Expenses', 'Manage business expenses', 'FileSpreadsheet', '/financials/expenses', '6'],

        ['Settings', 'business_profile', 'Business Profile', 'Manage business profile and details', 'Building2', '/settings/profile', '1'],
        ['Settings', 'business_tax_settings', 'Tax Settings', 'Configure tax rates and rules', 'ReceiptText', '/settings/tax', '2'],
        ['Settings', 'business_printer', 'Printer Setup', 'Configure printers and hardware', 'Printer', '/settings/printer', '3'],
        ['Settings', 'business_integrations', 'Integrations', 'Manage integrations', 'Plug', '/settings/integrations', '4'],
        ['Settings', 'business_notifications', 'Notifications', 'Configure business notifications', 'Bell', '/settings/notifications', '5']
    ];

    cashier_modules TEXT[][] := ARRAY[
        -- [category_name, code, name, description, icon, path, sort_order]
        ['POS Operations', 'cashier_pos', 'Point of Sale', 'Process customer sales', 'ShoppingCart', '/pos', '1'],
        ['POS Operations', 'cashier_returns', 'Returns', 'Process returns', 'RotateCcw', '/pos/returns', '2'],
        ['POS Operations', 'cashier_customers', 'Customers', 'View customer information', 'Users', '/pos/customers', '3'],
        ['POS Operations', 'cashier_inventory', 'Quick Stock Check', 'Check product availability', 'Package', '/pos/stock-check', '4']
    ];

    -- Sub-module arrays
    admin_sub_modules TEXT[][] := ARRAY[
        -- [module_code, sub_code, sub_name, description, icon, url, sort_order]
        ['admin_businesses', 'business_list', 'Business List', 'View all businesses', 'Store', '/businesses/list', '1'],
        ['admin_businesses', 'business_onboarding', 'Onboarding', 'Approve new businesses', 'CheckCircle', '/businesses/onboarding', '2'],
        ['admin_businesses', 'business_billing', 'Business Billing', 'Manage business billing', 'CreditCard', '/businesses/billing', '3'],

        ['admin_users', 'user_list', 'User List', 'View all users', 'Users', '/users/list', '1'],
        ['admin_users', 'user_roles', 'Role Management', 'Manage roles', 'Shield', '/users/roles', '2'],
        ['admin_users', 'user_permissions', 'Permissions', 'Configure permissions', 'Lock', '/users/permissions', '3'],

        ['admin_invoices', 'invoice_list', 'Invoice List', 'View all invoices', 'FileText', '/billing/invoices', '1'],
        ['admin_invoices', 'invoice_create', 'Create Invoice', 'Create new invoice', 'PlusCircle', '/billing/invoices/create', '2'],

        ['admin_tickets', 'ticket_list', 'Ticket List', 'View all tickets', 'Ticket', '/support/tickets', '1'],
        ['admin_tickets', 'ticket_create', 'Create Ticket', 'Create new support ticket', 'PlusCircle', '/support/tickets/create', '2']
    ];

    business_sub_modules TEXT[][] := ARRAY[
        -- [module_code, sub_code, sub_name, description, icon, url, sort_order]
        ['business_pos', 'new_sale', 'New Sale', 'Process a new sale', 'PlusCircle', '/pos/new-sale', '1'],
        ['business_pos', 'checkout', 'Checkout', 'Complete checkout', 'ShoppingCart', '/pos/checkout', '2'],
        ['business_pos', 'quick_scan', 'Quick Scan', 'Quick product scan', 'ScanLine', '/pos/quick-scan', '3'],

        ['business_products', 'product_list', 'Product List', 'View all products', 'Box', '/inventory/products', '1'],
        ['business_products', 'add_product', 'Add Product', 'Add a new product', 'PlusCircle', '/inventory/products/add', '2'],
        ['business_products', 'edit_product', 'Edit Product', 'Edit product details', 'PenLine', '/inventory/products/edit', '3'],
        ['business_products', 'product_categories', 'Categories', 'Manage categories', 'FolderTree', '/inventory/categories', '4'],

        ['business_stock', 'stock_levels', 'Stock Levels', 'View current stock levels', 'RefreshCw', '/inventory/stock', '1'],
        ['business_stock', 'stock_adjustments', 'Stock Adjustments', 'Adjust stock levels', 'PenLine', '/inventory/stock/adjustments', '2'],
        ['business_stock', 'stock_take', 'Stock Take', 'Perform stock take', 'ClipboardList', '/inventory/stock/take', '3'],

        ['business_customers', 'customer_list', 'Customer List', 'View all customers', 'Users', '/customers', '1'],
        ['business_customers', 'add_customer', 'Add Customer', 'Add a new customer', 'UserPlus', '/customers/add', '2'],
        ['business_customers', 'loyalty_points', 'Loyalty Points', 'Manage loyalty points', 'Gift', '/customers/loyalty', '3']
    ];
BEGIN
    SELECT id INTO admin_role_id FROM roles WHERE code = 'admin' LIMIT 1;
    SELECT id INTO business_role_id FROM roles WHERE code = 'business' LIMIT 1;
    SELECT id INTO cashier_role_id FROM roles WHERE code = 'cashier' LIMIT 1;

    FOR i IN 1..array_length(admin_modules, 1) LOOP
        INSERT INTO modules (id, code, name, description, icon, path, role_id, sort_order)
        VALUES (
            gen_random_uuid(),
            admin_modules[i][2],
            admin_modules[i][3],
            admin_modules[i][4],
            admin_modules[i][5],
            admin_modules[i][6],
            admin_role_id,
            admin_modules[i][7]::INTEGER
        )
        ON CONFLICT (code) DO NOTHING;
    END LOOP;

    FOR i IN 1..array_length(business_modules, 1) LOOP
        INSERT INTO modules (id, code, name, description, icon, path, role_id, sort_order)
        VALUES (
            gen_random_uuid(),
            business_modules[i][2],
            business_modules[i][3],
            business_modules[i][4],
            business_modules[i][5],
            business_modules[i][6],
            business_role_id,
            business_modules[i][7]::INTEGER
        )
        ON CONFLICT (code) DO NOTHING;
    END LOOP;

    FOR i IN 1..array_length(cashier_modules, 1) LOOP
        INSERT INTO modules (id, code, name, description, icon, path, role_id, sort_order)
        VALUES (
            gen_random_uuid(),
            cashier_modules[i][2],
            cashier_modules[i][3],
            cashier_modules[i][4],
            cashier_modules[i][5],
            cashier_modules[i][6],
            cashier_role_id,
            cashier_modules[i][7]::INTEGER
        )
        ON CONFLICT (code) DO NOTHING;
    END LOOP;

    FOR i IN 1..array_length(admin_sub_modules, 1) LOOP
        SELECT id INTO seeded_module_id
        FROM modules
        WHERE code = admin_sub_modules[i][1]
        LIMIT 1;

        IF seeded_module_id IS NOT NULL THEN
            INSERT INTO sub_modules (id, module_id, role_id, url, code, name, description, icon, sort_order)
            VALUES (
                gen_random_uuid(),
                seeded_module_id,
                admin_role_id,
                admin_sub_modules[i][6],
                admin_sub_modules[i][2],
                admin_sub_modules[i][3],
                admin_sub_modules[i][4],
                admin_sub_modules[i][5],
                admin_sub_modules[i][7]::INTEGER
            )
            ON CONFLICT (module_id, code) DO NOTHING;
        END IF;
    END LOOP;

    FOR i IN 1..array_length(business_sub_modules, 1) LOOP
        SELECT id INTO seeded_module_id
        FROM modules
        WHERE code = business_sub_modules[i][1]
        LIMIT 1;

        IF seeded_module_id IS NOT NULL THEN
            INSERT INTO sub_modules (id, module_id, role_id, url, code, name, description, icon, sort_order)
            VALUES (
                gen_random_uuid(),
                seeded_module_id,
                business_role_id,
                business_sub_modules[i][6],
                business_sub_modules[i][2],
                business_sub_modules[i][3],
                business_sub_modules[i][4],
                business_sub_modules[i][5],
                business_sub_modules[i][7]::INTEGER
            )
            ON CONFLICT (module_id, code) DO NOTHING;
        END IF;
    END LOOP;

    -- Assign business modules to each business.
    INSERT INTO user_modules (user_id, module_id, business_id, sub_module_id)
    SELECT NULL, m.id, bm.business_id, NULL
    FROM (SELECT DISTINCT business_id FROM business_managers) bm
    CROSS JOIN modules m
    JOIN roles r ON r.id = m.role_id
    WHERE r.code = 'business'
      AND m.is_active = TRUE
    ON CONFLICT DO NOTHING;

    INSERT INTO user_modules (user_id, module_id, business_id, sub_module_id)
    SELECT NULL, sm.module_id, bm.business_id, sm.id
    FROM (SELECT DISTINCT business_id FROM business_managers) bm
    CROSS JOIN sub_modules sm
    JOIN modules m ON m.id = sm.module_id
    JOIN roles r ON r.id = m.role_id
    WHERE r.code = 'business'
      AND sm.is_active = TRUE
    ON CONFLICT DO NOTHING;
END $$;
