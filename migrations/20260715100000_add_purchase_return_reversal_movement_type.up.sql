INSERT INTO inventory_movement_types (movement_type, movement_direction, description)
VALUES ('purchase_return_reversal', 'In', 'Reversal of a purchase return')
ON CONFLICT (movement_type) DO NOTHING;
