CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    role_id UUID NOT NULL
        REFERENCES roles(id)
        ON DELETE RESTRICT,

    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_role_id
    ON user_roles(role_id);

INSERT INTO user_roles (user_id, role_id)
SELECT users.id, users.role_id
FROM users
WHERE users.role_id IS NOT NULL
ON CONFLICT (user_id, role_id) DO NOTHING;
