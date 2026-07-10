-- 20260709090019_create_modules_seeder.down.sql
-- Rollback modules and sub-modules seeder

-- Delete only the seeded records
DELETE FROM sub_modules
WHERE code IN (
    'business_list', 'business_onboarding', 'business_billing',
    'user_list', 'user_roles', 'user_permissions',
    'invoice_list', 'invoice_create',
    'ticket_list', 'ticket_create',
    'new_sale', 'checkout', 'quick_scan',
    'product_list', 'add_product', 'edit_product', 'product_categories',
    'stock_levels', 'stock_adjustments', 'stock_take',
    'customer_list', 'add_customer', 'loyalty_points'
);

DELETE FROM modules
WHERE code IN (
    'admin_dashboard', 'admin_system_settings', 'admin_audit_logs', 'admin_notifications',
    'admin_feature_flags', 'admin_api_keys', 'admin_businesses', 'admin_business_onboarding',
    'admin_business_subscriptions', 'admin_business_profiles', 'admin_business_status',
    'admin_users', 'admin_roles', 'admin_permissions', 'admin_staff', 'admin_sessions',
    'admin_invoices', 'admin_payments', 'admin_revenue', 'admin_transactions',
    'admin_tickets', 'admin_complaints', 'admin_escalations', 'admin_knowledge_base',
    'admin_system_analytics', 'admin_business_performance', 'admin_active_users',
    'admin_usage_reports', 'admin_traffic_reports', 'admin_login_audit',
    'admin_failed_logins', 'admin_blocked_users', 'admin_security_policies', 'admin_mfa',
    'business_dashboard', 'business_pos', 'business_sales_history', 'business_returns',
    'business_orders', 'business_products', 'business_categories', 'business_stock',
    'business_suppliers', 'business_purchase_orders', 'business_inventory_reports',
    'business_customers', 'business_loyalty', 'business_customer_segments',
    'business_customer_feedback', 'business_staff', 'business_shifts',
    'business_staff_permissions', 'business_staff_performance', 'business_payroll',
    'business_payments', 'business_transactions', 'business_sales_reports',
    'business_profit_loss', 'business_tax', 'business_expenses',
    'business_profile', 'business_tax_settings', 'business_printer',
    'business_integrations', 'business_notifications',
    'cashier_pos', 'cashier_returns', 'cashier_customers', 'cashier_inventory'
);
