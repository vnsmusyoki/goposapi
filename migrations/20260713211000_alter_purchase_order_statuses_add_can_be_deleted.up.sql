ALTER TABLE purchase_order_statuses
ADD COLUMN IF NOT EXISTS can_be_deleted BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE purchase_order_statuses
SET can_be_deleted = CASE
    WHEN code IN ('draft', 'pending_approval', 'approved', 'cancelled') THEN TRUE
    ELSE FALSE
END;
