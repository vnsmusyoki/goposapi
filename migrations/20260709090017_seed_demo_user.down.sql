DELETE FROM user_roles
WHERE user_id IN (
    SELECT id FROM users WHERE email = 'admin@gmail.com'
);

DELETE FROM users
WHERE email = 'admin@gmail.com';
