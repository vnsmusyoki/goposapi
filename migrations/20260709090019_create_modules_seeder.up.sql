-- 20260709090019_create_modules_seeder.up.sql
-- Seed modules and sub-modules for the POS system

-- ==========================================
-- ADMIN MODULES (Platform-level control)
-- ==========================================

-- 1. Dashboard Module
INSERT INTO modules (id, code, name, description, icon, role_id, sort_order, is_active)
VALUES (
    gen_random_uuid(),
    'admin_dashboard',
    'Dashboard',
    'Platform overview and analytics across businesses',
    'LayoutDashboard',
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    1,
    true
);

-- 2. Businesses Module
INSERT INTO modules (id, code, name, description, icon, role_id, sort_order, is_active)
VALUES (
    gen_random_uuid(),
    'admin_businesses',
    'Businesses / Merchants',
    'Manage all businesses on the platform',
    'Store',
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    2,
    true
);

-- 3. Users & Roles Module
INSERT INTO modules (id, code, name, description, icon, role_id, sort_order, is_active)
VALUES (
    gen_random_uuid(),
    'admin_users_roles',
    'Users & Roles',
    'Manage admin users, roles, and permissions',
    'Users',
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    3,
    true
);

-- 4. Modules & Features Module
INSERT INTO modules (id, code, name, description, icon, role_id, sort_order, is_active)
VALUES (
    gen_random_uuid(),
    'admin_modules_features',
    'Modules & Features',
    'Control feature access and module assignments per plan',
    'Blocks',
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    4,
    true
);

-- 5. Support Module
INSERT INTO modules (id, code, name, description, icon, role_id, sort_order, is_active)
VALUES (
    gen_random_uuid(),
    'admin_support',
    'Support / Tickets',
    'Handle support tickets and complaints',
    'LifeBuoy',
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    5,
    true
);

-- 6. Reports Module
INSERT INTO modules (id, code, name, description, icon, role_id, sort_order, is_active)
VALUES (
    gen_random_uuid(),
    'admin_reports',
    'Reports',
    'Platform-wide reports and analytics',
    'BarChart3',
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    6,
    true
);

-- 7. System Settings Module
INSERT INTO modules (id, code, name, description, icon, role_id, sort_order, is_active)
VALUES (
    gen_random_uuid(),
    'admin_system_settings',
    'System Settings',
    'Platform configuration and system settings',
    'Settings',
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    7,
    true
);

-- ==========================================
-- BUSINESS MODULES (Merchant-level)
-- ==========================================

-- 8. Business Dashboard Module
INSERT INTO modules (id, code, name, description, icon, role_id, sort_order, is_active)
VALUES (
    gen_random_uuid(),
    'business_dashboard',
    'Dashboard',
    'Business sales summary and daily overview',
    'LayoutDashboard',
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    1,
    true
);

-- 9. Sales / POS Module
INSERT INTO modules (id, code, name, description, icon, role_id, sort_order, is_active)
VALUES (
    gen_random_uuid(),
    'business_sales',
    'Sales / POS',
    'Point of sale operations and sales management',
    'ShoppingCart',
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    2,
    true
);

-- 10. Inventory Module
INSERT INTO modules (id, code, name, description, icon, role_id, sort_order, is_active)
VALUES (
    gen_random_uuid(),
    'business_inventory',
    'Inventory',
    'Manage products, categories, and stock',
    'Package',
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    3,
    true
);

-- 11. Customers Module
INSERT INTO modules (id, code, name, description, icon, role_id, sort_order, is_active)
VALUES (
    gen_random_uuid(),
    'business_customers',
    'Customers',
    'Manage customer profiles and loyalty programs',
    'Users',
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    4,
    true
);

-- 12. Employees Module
INSERT INTO modules (id, code, name, description, icon, role_id, sort_order, is_active)
VALUES (
    gen_random_uuid(),
    'business_employees',
    'Employees / Staff',
    'Manage staff, shifts, and permissions',
    'Badge',
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    5,
    true
);

-- 13. Business Reports Module
INSERT INTO modules (id, code, name, description, icon, role_id, sort_order, is_active)
VALUES (
    gen_random_uuid(),
    'business_reports',
    'Reports',
    'Business sales, inventory, and profit reports',
    'PieChart',
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    6,
    true
);

-- 14. Payments Module
INSERT INTO modules (id, code, name, description, icon, role_id, sort_order, is_active)
VALUES (
    gen_random_uuid(),
    'business_payments',
    'Payments',
    'Manage payment methods and transaction history',
    'CreditCard',
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    7,
    true
);

-- 15. Business Settings Module
INSERT INTO modules (id, code, name, description, icon, role_id, sort_order, is_active)
VALUES (
    gen_random_uuid(),
    'business_settings',
    'Settings',
    'Business profile, tax settings, and hardware setup',
    'Settings2',
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    8,
    true
);

-- ==========================================
-- SUB-MODULES
-- ==========================================

-- Sub-modules for Businesses (admin_businesses)
INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    'business_onboarding',
    'Onboarding / Approvals',
    'Approve and onboard new businesses',
    'CheckCircle',
    1,
    true
FROM modules m
WHERE m.code = 'admin_businesses';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    'business_subscription',
    'Subscription & Billing',
    'Manage business subscriptions and billing',
    'CreditCard',
    2,
    true
FROM modules m
WHERE m.code = 'admin_businesses';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    'business_settings_admin',
    'Business Settings',
    'Configure business settings from admin side',
    'Sliders',
    3,
    true
FROM modules m
WHERE m.code = 'admin_businesses';

-- Sub-modules for Users & Roles (admin_users_roles)
INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    'admin_users_list',
    'Admin Users',
    'Manage platform admin users',
    'UserCog',
    1,
    true
FROM modules m
WHERE m.code = 'admin_users_roles';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    'role_management',
    'Role Management',
    'Create and manage user roles',
    'Shield',
    2,
    true
FROM modules m
WHERE m.code = 'admin_users_roles';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    'permissions_management',
    'Permissions',
    'Configure role-based permissions',
    'Key',
    3,
    true
FROM modules m
WHERE m.code = 'admin_users_roles';

-- Sub-modules for Modules & Features (admin_modules_features)
INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    'module_assignment',
    'Module Assignment',
    'Assign modules to roles and businesses',
    'Grid',
    1,
    true
FROM modules m
WHERE m.code = 'admin_modules_features';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    'feature_flags',
    'Feature Flags',
    'Manage feature toggles and flags',
    'Flag',
    2,
    true
FROM modules m
WHERE m.code = 'admin_modules_features';

-- Sub-modules for Support (admin_support)
INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    'complaints',
    'Complaints',
    'Manage user complaints and issues',
    'MessageSquare',
    1,
    true
FROM modules m
WHERE m.code = 'admin_support';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    'refund_requests',
    'Refund Requests',
    'Process refund requests',
    'Undo2',
    2,
    true
FROM modules m
WHERE m.code = 'admin_support';

-- Sub-modules for Reports (admin_reports)
INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    'revenue_reports',
    'Revenue Reports',
    'Platform revenue reports and analytics',
    'TrendingUp',
    1,
    true
FROM modules m
WHERE m.code = 'admin_reports';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    'usage_reports',
    'Usage Reports',
    'Platform usage statistics and reports',
    'Activity',
    2,
    true
FROM modules m
WHERE m.code = 'admin_reports';

-- Sub-modules for System Settings (admin_system_settings)
INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    'notifications_admin',
    'Notifications',
    'Configure platform notifications',
    'Bell',
    1,
    true
FROM modules m
WHERE m.code = 'admin_system_settings';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    'integrations',
    'Integrations',
    'Manage payment gateways, SMS, and other integrations',
    'Plug',
    2,
    true
FROM modules m
WHERE m.code = 'admin_system_settings';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code ='admin' LIMIT 1),
    'audit_logs',
    'Audit Logs',
    'View platform audit logs and activities',
    'ClipboardList',
    3,
    true
FROM modules m
WHERE m.code = 'admin_system_settings';

-- Sub-modules for Sales / POS (business_sales)
INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'new_sale',
    'New Sale / Checkout',
    'Process new sales and checkout',
    'PlusCircle',
    1,
    true
FROM modules m
WHERE m.code = 'business_sales';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'sales_history',
    'Sales History',
    'View sales history and past transactions',
    'History',
    2,
    true
FROM modules m
WHERE m.code = 'business_sales';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'returns_refunds',
    'Returns & Refunds',
    'Process returns and refunds',
    'RotateCcw',
    3,
    true
FROM modules m
WHERE m.code = 'business_sales';

-- Sub-modules for Inventory (business_inventory)
INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'products',
    'Products',
    'Manage product catalog',
    'Box',
    1,
    true
FROM modules m
WHERE m.code = 'business_inventory';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'categories',
    'Categories',
    'Manage product categories',
    'FolderTree',
    2,
    true
FROM modules m
WHERE m.code = 'business_inventory';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'stock_adjustments',
    'Stock Adjustments',
    'Manage stock adjustments and inventory counts',
    'RefreshCw',
    3,
    true
FROM modules m
WHERE m.code = 'business_inventory';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'suppliers',
    'Suppliers',
    'Manage suppliers and purchase orders',
    'Truck',
    4,
    true
FROM modules m
WHERE m.code = 'business_inventory';

-- Sub-modules for Customers (business_customers)
INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'customer_list',
    'Customer List',
    'View and manage customer profiles',
    'Users',
    1,
    true
FROM modules m
WHERE m.code = 'business_customers';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'loyalty_points',
    'Loyalty / Points',
    'Manage customer loyalty programs and points',
    'Gift',
    2,
    true
FROM modules m
WHERE m.code = 'business_customers';

-- Sub-modules for Employees (business_employees)
INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'staff_list',
    'Staff List',
    'Manage staff members',
    'Users',
    1,
    true
FROM modules m
WHERE m.code = 'business_employees';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'shifts_attendance',
    'Shifts / Attendance',
    'Manage staff shifts and attendance',
    'Calendar',
    2,
    true
FROM modules m
WHERE m.code = 'business_employees';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'staff_permissions',
    'Permissions (cashier vs manager)',
    'Set staff roles and permissions',
    'Lock',
    3,
    true
FROM modules m
WHERE m.code = 'business_employees';

-- Sub-modules for Business Reports (business_reports)
INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'sales_reports_business',
    'Sales Reports',
    'View sales reports and analytics',
    'TrendingUp',
    1,
    true
FROM modules m
WHERE m.code = 'business_reports';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'inventory_reports',
    'Inventory Reports',
    'View inventory and stock reports',
    'Package',
    2,
    true
FROM modules m
WHERE m.code = 'business_reports';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'profit_loss',
    'Profit & Loss',
    'View profit and loss reports',
    'DollarSign',
    3,
    true
FROM modules m
WHERE m.code = 'business_reports';

-- Sub-modules for Payments (business_payments)
INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'payment_methods',
    'Payment Methods',
    'Configure payment methods',
    'CreditCard',
    1,
    true
FROM modules m
WHERE m.code = 'business_payments';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'transaction_history',
    'Transaction History',
    'View all transaction history',
    'Receipt',
    2,
    true
FROM modules m
WHERE m.code = 'business_payments';

-- Sub-modules for Business Settings (business_settings)
INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'business_profile',
    'Business Profile',
    'Manage business profile and details',
    'Building2',
    1,
    true
FROM modules m
WHERE m.code = 'business_settings';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'tax_receipt_settings',
    'Tax & Receipt Settings',
    'Configure tax rates and receipt templates',
    'ReceiptText',
    2,
    true
FROM modules m
WHERE m.code = 'business_settings';

INSERT INTO sub_modules (module_id, role_id, code, name, description, icon, sort_order, is_active)
SELECT 
    m.id,
    (SELECT id FROM roles WHERE code = 'business' LIMIT 1),
    'printer_hardware',
    'Printer / Hardware Setup',
    'Configure printers and hardware devices',
    'Printer',
    3,
    true
FROM modules m
WHERE m.code = 'business_settings';
