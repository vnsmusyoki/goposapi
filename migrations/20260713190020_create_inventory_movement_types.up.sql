CREATE TABLE IF NOT EXISTS inventory_movement_types (
    movement_type VARCHAR(50) PRIMARY KEY,
    movement_direction VARCHAR(3) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT inventory_movement_types_direction_check
        CHECK (movement_direction IN ('In', 'Out'))
);

CREATE TRIGGER set_inventory_movement_types_updated_at
BEFORE UPDATE ON inventory_movement_types
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

INSERT INTO inventory_movement_types (movement_type, movement_direction, description)
VALUES
    ('opening_stock', 'In', 'Initial stock setup'),
    ('purchase_receipt', 'In', 'Goods received from supplier'),
    ('purchase_return', 'Out', 'Items returned to supplier'),
    ('sale', 'Out', 'Customer purchase'),
    ('sale_return', 'In', 'Customer returns goods'),
    ('stock_transfer_in', 'In', 'Stock received from another branch'),
    ('stock_transfer_out', 'Out', 'Stock sent to another branch'),
    ('adjustment_in', 'In', 'Physical count found extra stock'),
    ('adjustment_out', 'Out', 'Physical count found missing stock'),
    ('damage', 'Out', 'Damaged goods removed'),
    ('expired', 'Out', 'Expired goods removed'),
    ('write_off', 'Out', 'Lost, stolen, or unusable stock'),
    ('production_in', 'In', 'Manufactured/assembled product added'),
    ('production_out', 'Out', 'Raw materials consumed in production')
ON CONFLICT (movement_type) DO NOTHING;
