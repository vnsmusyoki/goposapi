INSERT INTO users (
    business_id,
    store_id,
    username,
    password_hash,
    email,
    full_name,
    phone,
    role_id,
    is_active
)
VALUES (
    NULL,
    NULL,
    'admin',
    crypt('Password@123', gen_salt('bf')),
    'admin@gmail.com',
    'Admin Account',
    NULL,
    (SELECT id FROM roles WHERE code = 'admin'),
    TRUE
)
ON CONFLICT (email) DO UPDATE
SET
    password_hash = EXCLUDED.password_hash,
    full_name = EXCLUDED.full_name,
    role_id = EXCLUDED.role_id,
    business_id = EXCLUDED.business_id,
    is_active = TRUE;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u
JOIN roles r ON r.code = 'admin'
WHERE u.email = 'admin@gmail.com'
ON CONFLICT DO NOTHING;


-- =========================================================
-- Assign ALL admin-role modules to the admin user
-- =========================================================

-- 1) Modules with no sub-modules (e.g. Home) -> module-level row, sub_module_id NULL
INSERT INTO user_modules (user_id, module_id, business_id, sub_module_id)
SELECT
    u.id,
    m.id,
    NULL,
    NULL
FROM users u
JOIN roles r   ON r.code = 'admin'
JOIN modules m ON m.role_id = r.id
WHERE u.email = 'admin@gmail.com'
  AND m.has_sub_modules = FALSE
ON CONFLICT (user_id, module_id, sub_module_id, business_id) DO NOTHING;

-- 2) Modules that DO have sub-modules -> also give the parent module-level row
--    (useful if you want the nav item itself assigned, not just its children)
INSERT INTO user_modules (user_id, module_id, business_id, sub_module_id)
SELECT
    u.id,
    m.id,
    NULL,
    NULL
FROM users u
JOIN roles r   ON r.code = 'admin'
JOIN modules m ON m.role_id = r.id
WHERE u.email = 'admin@gmail.com'
  AND m.has_sub_modules = TRUE
ON CONFLICT (user_id, module_id, sub_module_id, business_id) DO NOTHING;

-- 3) Every sub-module under an admin-role module -> one row per sub-module
INSERT INTO user_modules (user_id, module_id, business_id, sub_module_id)
SELECT
    u.id,
    sm.module_id,
    NULL,
    sm.id
FROM users u
JOIN roles r        ON r.code = 'admin'
JOIN modules m       ON m.role_id = r.id
JOIN sub_modules sm  ON sm.module_id = m.id
WHERE u.email = 'admin@gmail.com'
ON CONFLICT (user_id, module_id, sub_module_id, business_id) DO NOTHING;