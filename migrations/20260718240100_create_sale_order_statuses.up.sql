CREATE TABLE IF NOT EXISTS sale_order_statuses (
    code VARCHAR(30) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    what_happens TEXT NOT NULL DEFAULT '',
    requires_further_action BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_sale_order_statuses_updated_at
BEFORE UPDATE ON sale_order_statuses
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

INSERT INTO sale_order_statuses (code, name, what_happens, requires_further_action, sort_order) VALUES
('draft', 'Draft', 'Order is being prepared and can still be changed.', TRUE, 1),
('pending_approval', 'Pending Approval', 'Order is waiting for approval and can still move forward.', TRUE, 2),
('approved', 'Approved', 'Order has been approved and can continue through the sales workflow.', TRUE, 3),
('processing', 'Processing', 'Order is being worked on and stock may be reserved or prepared.', TRUE, 4),
('ready_for_shipment', 'Ready for Shipment', 'Order is ready for loading and no further workflow action is needed.', FALSE, 5),
('completed', 'Completed', 'Order is finalized and cannot move backward.', FALSE, 6)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    what_happens = EXCLUDED.what_happens,
    requires_further_action = EXCLUDED.requires_further_action,
    sort_order = EXCLUDED.sort_order,
    updated_at = CURRENT_TIMESTAMP;

ALTER TABLE sales_orders
    ADD COLUMN IF NOT EXISTS requires_further_action BOOLEAN NOT NULL DEFAULT TRUE;

UPDATE sales_orders so
SET requires_further_action = s.requires_further_action
FROM sale_order_statuses s
WHERE s.code = so.status;

ALTER TABLE sales_orders
    ADD CONSTRAINT fk_sales_orders_status
    FOREIGN KEY (status) REFERENCES sale_order_statuses(code)
    ON UPDATE CASCADE
    ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_sale_order_statuses_sort_order
    ON sale_order_statuses (sort_order);

CREATE INDEX IF NOT EXISTS idx_sales_orders_status
    ON sales_orders (status);
