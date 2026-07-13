CREATE TABLE IF NOT EXISTS purchase_order_statuses (
    code VARCHAR(30) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    what_happens TEXT NOT NULL DEFAULT '',
    editable_note VARCHAR(50) NOT NULL DEFAULT '',
    stock_affected_note VARCHAR(50) NOT NULL DEFAULT '',
    requires_receiving_items BOOLEAN NOT NULL DEFAULT FALSE,
    prepare_invoice BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_purchase_order_statuses_updated_at
BEFORE UPDATE ON purchase_order_statuses
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

INSERT INTO purchase_order_statuses (code, name, what_happens, editable_note, stock_affected_note, requires_receiving_items, prepare_invoice, sort_order) VALUES
('draft', 'Draft', 'Order is being prepared. Items, supplier, prices, and quantities can still be changed.', 'Yes', 'No', FALSE, FALSE, 1),
('pending_approval', 'Pending Approval', 'Order has been submitted and is waiting for approval.', 'Usually no', 'No', FALSE, FALSE, 2),
('approved', 'Approved', 'A manager has approved the order and it is ready to be placed with the supplier.', 'Limited', 'No', FALSE, FALSE, 3),
('ordered', 'Ordered', 'Order has been sent or placed with the supplier.', 'No', 'No', FALSE, FALSE, 4),
('partially_received', 'Partially Received', 'Some ordered items have arrived. Only the received quantities should be added to stock.', 'No', 'Yes', TRUE, FALSE, 5),
('received', 'Received', 'All expected items have been received and added to stock.', 'No', 'Yes', TRUE, TRUE, 6),
('completed', 'Completed', 'Purchasing has been finalized and no further action is expected.', 'No', 'No additional stock', TRUE, TRUE, 7),
('cancelled', 'Cancelled', 'Order has been stopped and no further deliveries are expected.', 'No', 'No', FALSE, FALSE, 8),
('closed', 'Closed', 'Order is locked after everything has been finalized.', 'No', 'Yes', FALSE, TRUE, 9)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    what_happens = EXCLUDED.what_happens,
    editable_note = EXCLUDED.editable_note,
    stock_affected_note = EXCLUDED.stock_affected_note,
    requires_receiving_items = EXCLUDED.requires_receiving_items,
    prepare_invoice = EXCLUDED.prepare_invoice,
    sort_order = EXCLUDED.sort_order,
    updated_at = CURRENT_TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_purchase_order_statuses_sort_order
    ON purchase_order_statuses (sort_order);
