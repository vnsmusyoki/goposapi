INSERT INTO businesses (
    name,
    business_email,
    business_phone,
    registration_number,
    industry,
    owner_name,
    subscription_plan,
    is_active
)
VALUES (
    'FlowPOS Demo Business',
    'demo@flowpos.local',
    NULL,
    NULL,
    'Retail',
    'Demo Owner',
    'free',
    TRUE
)
ON CONFLICT (business_email) DO NOTHING;

INSERT INTO users (
    business_id,
    store_id,
    username,
    password_hash,
    email,
    full_name,
    phone,
    role,
    is_active
)
SELECT
    b.id,
    NULL,
    'demo.admin',
    crypt('Password123!', gen_salt('bf')),
    'admin@flowpos.local',
    'Demo Admin',
    NULL,
    'admin',
    TRUE
FROM businesses b
WHERE b.business_email = 'demo@flowpos.local'
ON CONFLICT (email) DO UPDATE
SET
    password_hash = EXCLUDED.password_hash,
    full_name = EXCLUDED.full_name,
    role = EXCLUDED.role,
    business_id = EXCLUDED.business_id,
    is_active = TRUE;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u
JOIN roles r ON r.code = 'admin'
WHERE u.email = 'admin@flowpos.local'
ON CONFLICT DO NOTHING;
