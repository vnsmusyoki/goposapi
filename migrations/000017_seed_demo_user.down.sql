DELETE FROM user_roles
WHERE user_id IN (
    SELECT id FROM users WHERE email = 'admin@flowpos.local'
);

DELETE FROM users
WHERE email = 'admin@flowpos.local';

DELETE FROM businesses
WHERE business_email = 'demo@flowpos.local';
