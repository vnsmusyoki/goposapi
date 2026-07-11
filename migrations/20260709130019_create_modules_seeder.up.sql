-- =========================================================
-- ADMIN ROLE MODULES
-- =========================================================
INSERT INTO modules (code, name, description, icon, path, has_sub_modules, access_level, role_id, sort_order, is_active)
VALUES
('admin-home',          'Home',               'Admin home dashboard', 'home',    'home',               FALSE, 1, (SELECT id FROM roles WHERE code =  'admin'), 1, TRUE),
('business-management', 'Business Management','Manage businesses',    'briefcase','business-management', TRUE,  1, (SELECT id FROM roles WHERE code =  'admin'), 2, TRUE),
('module-management',   'Module Management',  'Manage modules',       'layers',  'module-management',   TRUE,  1, (SELECT id FROM roles WHERE code =  'admin'), 3, TRUE);

-- Sub modules: Business Management
INSERT INTO sub_modules (module_id, role_id, url, code, name, description, icon, access_level, sort_order, is_active)
VALUES
((SELECT id FROM modules WHERE code = 'business-management'), (SELECT id FROM roles WHERE code =  'admin'),
 'business-management/list', 'list-businesses', 'List Businesses', 'View all businesses', 'list', 1, 1, TRUE),
((SELECT id FROM modules WHERE code = 'business-management'), (SELECT id FROM roles WHERE code =  'admin'),
 'business-management/create',  'add-business',    'Add Business',    'Add a new business',  'plus', 1, 2, TRUE);

-- Sub modules: Module Management
INSERT INTO sub_modules (module_id, role_id, url, code, name, description, icon, access_level, sort_order, is_active)
VALUES
((SELECT id FROM modules WHERE code = 'module-management'), (SELECT id FROM roles WHERE code =  'admin'),
 'module-management/list', 'list-modules', 'List Modules', 'View all modules',  'list', 1, 1, TRUE),
((SELECT id FROM modules WHERE code = 'module-management'), (SELECT id FROM roles WHERE code =  'admin'),
 'module-management/add',  'add-modules',  'Add Modules',  'Add a new module', 'plus', 1, 2, TRUE);


-- =========================================================
-- BUSINESS ROLE MODULES
-- =========================================================
INSERT INTO modules (code, name, description, icon, path, has_sub_modules, access_level, role_id, sort_order, is_active)
VALUES
('business-home',   'Home',            'Business home dashboard',        'home',  'home',            FALSE, 1, (SELECT id FROM roles WHERE code =  'business'), 1, TRUE),
('user-management', 'User Management', 'Manage users, roles and agents', 'users', 'user-management',  TRUE,  1, (SELECT id FROM roles WHERE code =  'business'), 2, TRUE);

-- Sub modules: User Management
INSERT INTO sub_modules (module_id, role_id, url, code, name, description, icon, access_level, sort_order, is_active)
VALUES
((SELECT id FROM modules WHERE code = 'user-management'), (SELECT id FROM roles WHERE code =  'business'),
 'user-management/users',                     'users',                    'Users',                    'Manage users',                    'user',    1, 1, TRUE),
((SELECT id FROM modules WHERE code = 'user-management'), (SELECT id FROM roles WHERE code =  'business'),
 'user-management/roles',                     'roles',                    'Role',                     'Manage roles',                     'shield',  1, 2, TRUE),
((SELECT id FROM modules WHERE code = 'user-management'), (SELECT id FROM roles WHERE code =  'business'),
 'user-management/sales-commission-agents',   'sales-commission-agents',  'Sales Commission Agents',  'Manage sales commission agents',  'percent', 1, 3, TRUE);