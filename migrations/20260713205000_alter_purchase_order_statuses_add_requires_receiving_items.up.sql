ALTER TABLE purchase_order_statuses
ADD COLUMN IF NOT EXISTS requires_receiving_items BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE purchase_order_statuses
SET requires_receiving_items = CASE
    WHEN code IN ('partially_received', 'received', 'completed') THEN TRUE
    ELSE FALSE
END;
