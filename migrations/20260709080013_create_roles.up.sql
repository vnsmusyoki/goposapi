CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,

    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_roles_updated_at
BEFORE UPDATE ON roles
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

INSERT INTO roles (code, name, description, sort_order)
VALUES
    ('admin', 'Admin', 'Full system access', 1),
    ('business', 'Business', 'Business owner access', 2),
    ('owner', 'Owner', 'Business owner access', 3),
    ('manager', 'Manager', 'Managerial access', 4),
    ('cashier', 'Cashier', 'Point-of-sale and cashier access', 5),
    ('staff', 'Staff', 'General staff access', 6)
ON CONFLICT (code) DO NOTHING;
