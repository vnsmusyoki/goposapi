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
