CREATE TABLE IF NOT EXISTS modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    icon VARCHAR(100),
    path VARCHAR(255),
    role_id UUID NOT NULL
        REFERENCES roles(id)
        ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_modules_updated_at
BEFORE UPDATE ON modules
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS sub_modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id UUID NOT NULL
        REFERENCES modules(id)
        ON DELETE CASCADE,
    role_id UUID NOT NULL
        REFERENCES roles(id)
        ON DELETE CASCADE,
    code VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    icon VARCHAR(100),
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (module_id, code)
);

CREATE TRIGGER set_sub_modules_updated_at
BEFORE UPDATE ON sub_modules
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS user_modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID  NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    module_id UUID NOT NULL
        REFERENCES modules(id)
        ON DELETE CASCADE,
    business_id UUID  NULL
        REFERENCES businesses(id)
        ON DELETE CASCADE,
    sub_module_id UUID NULL
        REFERENCES sub_modules(id)
        ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, module_id, sub_module_id, business_id)
);