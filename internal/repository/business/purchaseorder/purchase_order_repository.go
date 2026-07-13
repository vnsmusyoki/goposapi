package purchaseorder

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func ListPurchaseOrdersRepository(pool *pgxpool.Pool, businessID string) ([]PurchaseOrder, error) {
	return ListPurchaseOrdersWithFiltersRepository(pool, businessID, ListPurchaseOrdersFilters{})
}

func ListPurchaseOrdersWithFiltersRepository(pool *pgxpool.Pool, businessID string, filters ListPurchaseOrdersFilters) ([]PurchaseOrder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	query, args := purchaseOrderListQuery(businessID, filters)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list purchase orders: %w", err)
	}
	defer rows.Close()

	orders := make([]PurchaseOrder, 0)
	for rows.Next() {
		order, err := scanPurchaseOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purchase orders: %w", err)
	}

	return orders, nil
}

func GetPurchaseOrderByIDRepository(pool *pgxpool.Pool, businessID, purchaseOrderID string) (*PurchaseOrder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	purchaseOrderID = strings.TrimSpace(purchaseOrderID)
	if businessID == "" || purchaseOrderID == "" {
		return nil, ErrBusinessNotResolved
	}

	return getPurchaseOrderByID(ctx, pool, businessID, purchaseOrderID)
}

func CreatePurchaseOrderRepository(pool *pgxpool.Pool, req CreatePurchaseOrderInput) (*PurchaseOrder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.SupplierID = strings.TrimSpace(req.SupplierID)
	req.LocationID = strings.TrimSpace(req.LocationID)
	req.ReferenceNumber = strings.TrimSpace(req.ReferenceNumber)
	req.OrderDate = strings.TrimSpace(req.OrderDate)
	req.DeliveryDate = strings.TrimSpace(req.DeliveryDate)
	req.PaymentTermUnit = strings.ToLower(strings.TrimSpace(req.PaymentTermUnit))
	req.AttachmentName = strings.TrimSpace(req.AttachmentName)
	req.AttachmentURL = strings.TrimSpace(req.AttachmentURL)
	req.Notes = strings.TrimSpace(req.Notes)
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	req.DeliveryStatus = strings.ToLower(strings.TrimSpace(req.DeliveryStatus))
	req.PaymentStatus = strings.ToLower(strings.TrimSpace(req.PaymentStatus))
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)

	if req.BusinessID == "" || req.SupplierID == "" || req.LocationID == "" || req.OrderDate == "" {
		return nil, ErrInvalidPurchaseOrderInput
	}

	if req.PaymentTermUnit == "" {
		req.PaymentTermUnit = "days"
	}
	if req.Status == "" {
		req.Status = "draft"
	}
	if req.DeliveryStatus == "" {
		req.DeliveryStatus = "pending_delivery"
	}
	if req.PaymentStatus == "" {
		req.PaymentStatus = "unpaid"
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin purchase order tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	row := tx.QueryRow(ctx, `
		INSERT INTO purchase_orders (
			business_id,
			supplier_id,
			location_id,
			reference_number,
			order_date,
			delivery_date,
			payment_term_value,
			payment_term_unit,
			attachment_name,
			attachment_url,
			delivery_address,
			delivery_charges,
			delivery_document_name,
			delivery_document_url,
			order_discount_amount,
			notes,
			status,
			delivery_status,
			payment_status,
			subtotal,
			total_discount,
			total_tax,
			grand_total,
			items_count,
			total_quantity,
			created_by
		) VALUES (
			$1,
			$2::uuid,
			$3::uuid,
			$4,
			$5::date,
			NULLIF($6, '')::date,
			$7,
			$8,
			NULLIF($9, ''),
			NULLIF($10, ''),
			NULLIF($11, ''),
			$12,
			NULLIF($13, ''),
			NULLIF($14, ''),
			$15,
			$16,
			$17,
			$18,
			$19,
			$20,
			$21,
			$22,
			$23,
			$24,
			$25,
			NULLIF($26, '')::uuid
		)
		RETURNING id::text
	`, req.BusinessID, req.SupplierID, req.LocationID, req.ReferenceNumber, req.OrderDate, req.DeliveryDate, req.PaymentTermValue, req.PaymentTermUnit, req.AttachmentName, req.AttachmentURL, req.DeliveryAddress, req.DeliveryCharges, req.DeliveryDocument, "", req.OrderDiscountAmount, req.Notes, req.Status, req.DeliveryStatus, req.PaymentStatus, req.Subtotal, req.TotalDiscount, req.TotalTax, req.GrandTotal, req.ItemsCount, req.TotalQuantity, req.CreatedBy)

	var purchaseOrderID string
	if err := row.Scan(&purchaseOrderID); err != nil {
		return nil, fmt.Errorf("insert purchase order: %w", err)
	}

	for idx, item := range req.Items {
		var productName, sku, unitName string
		if err := tx.QueryRow(ctx, `
			SELECT
				p.name,
				COALESCE(p.sku, ''),
				COALESCE(u.name, '')
			FROM products p
			LEFT JOIN business_units u ON u.id = p.unit_id
			WHERE p.business_id = $1
			  AND p.id = $2::uuid
			  AND p.deleted_at IS NULL
			LIMIT 1
		`, req.BusinessID, item.ProductID).Scan(&productName, &sku, &unitName); err != nil {
			if err == pgx.ErrNoRows {
				return nil, fmt.Errorf("purchase order item product not found")
			}
			return nil, fmt.Errorf("resolve purchase order item product: %w", err)
		}

		item.PurchaseOrderID = purchaseOrderID
		if strings.TrimSpace(productName) == "" {
			productName = strings.TrimSpace(item.ProductID)
		}
		if strings.TrimSpace(sku) == "" {
			sku = ""
		}
		if strings.TrimSpace(unitName) == "" {
			unitName = ""
		}

		receivedQuantity := 0.0
		if item.ReceivedQuantity != nil {
			receivedQuantity = *item.ReceivedQuantity
		}
		if receivedQuantity < 0 {
			receivedQuantity = 0
		}
		if receivedQuantity > item.OrderQuantity {
			receivedQuantity = item.OrderQuantity
		}
		balanceQuantity := item.OrderQuantity - receivedQuantity
		receivedStatus := "pending"
		if receivedQuantity > 0 && balanceQuantity > 0 {
			receivedStatus = "partial"
		} else if balanceQuantity <= 0 {
			receivedStatus = "received"
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO purchase_order_items (
				purchase_order_id,
				business_id,
				product_id,
				product_name,
				sku,
				unit,
				order_quantity,
				unit_cost_before_discount,
				discount_percentage,
				discount_amount,
				unit_cost_before_tax,
				product_tax_rate,
				tax_amount,
				net_cost,
				selling_price,
				line_cost,
				manufacture_date,
				expiry_date,
				lot_number,
				balance_quantity,
				received_quantity,
				items_received,
				received_status,
				sort_order
			) VALUES (
				NULLIF($1, '')::uuid,
				$2::uuid,
				$3::uuid,
				$4,
				$5,
				$6,
				$7,
				$8,
				$9,
				$10,
				$11,
				$12,
				$13,
				$14,
				$15,
				$16,
				NULLIF($17, '')::date,
				NULLIF($18, '')::date,
				$19,
				$21,
				$20,
				$20,
				$22,
				$23
			)
		`, item.PurchaseOrderID, req.BusinessID, item.ProductID, productName, sku, unitName, item.OrderQuantity, item.UnitCostBeforeDiscount, item.DiscountPercentage, item.DiscountAmount, item.UnitCostBeforeTax, item.ProductTaxRate, item.TaxAmount, item.NetCost, item.SellingPrice, item.LineCost, item.ManufactureDate, item.ExpiryDate, item.LotNumber, receivedQuantity, balanceQuantity, receivedStatus, idx); err != nil {
			return nil, fmt.Errorf("insert purchase order item: %w", err)
		}
	}

	for _, expense := range req.AdditionalExpenses {
		if _, err := tx.Exec(ctx, `
			INSERT INTO purchase_order_additional_expenses (
				purchase_order_id,
				business_id,
				expense_name,
				amount,
				sort_order
			) VALUES (
				NULLIF($1, '')::uuid,
				$2::uuid,
				$3,
				$4,
				$5
			)
		`, purchaseOrderID, req.BusinessID, expense.Name, expense.Amount, expense.SortOrder); err != nil {
			return nil, fmt.Errorf("insert purchase order additional expense: %w", err)
		}
	}

	created, err := getPurchaseOrderByID(ctx, tx, req.BusinessID, purchaseOrderID)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(req.ActivityAction) != "" && strings.TrimSpace(req.ActivityNote) != "" {
		if err := insertPurchaseOrderLog(ctx, tx, CreatePurchaseOrderLogInput{
			BusinessID:      req.BusinessID,
			PurchaseOrderID: purchaseOrderID,
			Action:          req.ActivityAction,
			ActionedBy:      req.ActivityActionedBy,
			Note:            req.ActivityNote,
		}); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit purchase order tx: %w", err)
	}

	return created, nil
}

func UpdatePurchaseOrderRepository(pool *pgxpool.Pool, req UpdatePurchaseOrderInput) (*PurchaseOrder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.PurchaseOrderID = strings.TrimSpace(req.PurchaseOrderID)
	req.SupplierID = strings.TrimSpace(req.SupplierID)
	req.LocationID = strings.TrimSpace(req.LocationID)
	req.ReferenceNumber = strings.TrimSpace(req.ReferenceNumber)
	req.OrderDate = strings.TrimSpace(req.OrderDate)
	req.DeliveryDate = strings.TrimSpace(req.DeliveryDate)
	req.PaymentTermUnit = strings.ToLower(strings.TrimSpace(req.PaymentTermUnit))
	req.AttachmentName = strings.TrimSpace(req.AttachmentName)
	req.AttachmentURL = strings.TrimSpace(req.AttachmentURL)
	req.DeliveryAddress = strings.TrimSpace(req.DeliveryAddress)
	req.DeliveryDocument = strings.TrimSpace(req.DeliveryDocument)
	req.Notes = strings.TrimSpace(req.Notes)
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	req.DeliveryStatus = strings.ToLower(strings.TrimSpace(req.DeliveryStatus))
	req.PaymentStatus = strings.ToLower(strings.TrimSpace(req.PaymentStatus))
	req.UpdatedBy = strings.TrimSpace(req.UpdatedBy)
	req.PreviousStatus = strings.ToLower(strings.TrimSpace(req.PreviousStatus))
	req.PreviousDeliveryStatus = strings.ToLower(strings.TrimSpace(req.PreviousDeliveryStatus))
	req.PreviousPaymentStatus = strings.ToLower(strings.TrimSpace(req.PreviousPaymentStatus))

	if req.BusinessID == "" || req.PurchaseOrderID == "" || req.SupplierID == "" || req.LocationID == "" || req.OrderDate == "" {
		return nil, ErrInvalidPurchaseOrderInput
	}

	if req.PaymentTermUnit == "" {
		req.PaymentTermUnit = "days"
	}
	if req.Status == "" {
		req.Status = "draft"
	}
	if req.DeliveryStatus == "" {
		req.DeliveryStatus = "pending_delivery"
	}
	if req.PaymentStatus == "" {
		req.PaymentStatus = "unpaid"
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin purchase order update tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	existing, err := getPurchaseOrderByID(ctx, tx, req.BusinessID, req.PurchaseOrderID)
	if err != nil {
		return nil, err
	}

	commandTag, err := tx.Exec(ctx, `
		UPDATE purchase_orders
		SET supplier_id = $3::uuid,
		    location_id = $4::uuid,
		    reference_number = $5,
		    order_date = $6::date,
		    delivery_date = NULLIF($7, '')::date,
		    payment_term_value = $8,
		    payment_term_unit = $9,
		    attachment_name = NULLIF($10, ''),
		    attachment_url = NULLIF($11, ''),
		    delivery_address = NULLIF($12, ''),
		    delivery_charges = $13,
		    delivery_document_name = NULLIF($14, ''),
		    delivery_document_url = NULLIF($15, ''),
		    order_discount_amount = $16,
		    notes = $17,
		    status = $18,
		    delivery_status = $19,
		    payment_status = $20,
		    subtotal = $21,
		    total_discount = $22,
		    total_tax = $23,
		    grand_total = $24,
		    items_count = $25,
		    total_quantity = $26,
		    updated_at = CURRENT_TIMESTAMP
		WHERE business_id = $1
		  AND id = $2::uuid
		  AND deleted_at IS NULL
	`, req.BusinessID, req.PurchaseOrderID, req.SupplierID, req.LocationID, req.ReferenceNumber, req.OrderDate, req.DeliveryDate, req.PaymentTermValue, req.PaymentTermUnit, req.AttachmentName, req.AttachmentURL, req.DeliveryAddress, req.DeliveryCharges, req.DeliveryDocument, "", req.OrderDiscountAmount, req.Notes, req.Status, req.DeliveryStatus, req.PaymentStatus, req.Subtotal, req.TotalDiscount, req.TotalTax, req.GrandTotal, req.ItemsCount, req.TotalQuantity)
	if err != nil {
		return nil, fmt.Errorf("update purchase order: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return nil, ErrPurchaseOrderNotFound
	}

	existingItemReceipts := make(map[string]float64)
	rows, err := tx.Query(ctx, `
		SELECT
			product_id::text,
			COALESCE(received_quantity, 0)
		FROM purchase_order_items
		WHERE business_id = $1
		  AND purchase_order_id = $2::uuid
		  AND deleted_at IS NULL
	`, req.BusinessID, req.PurchaseOrderID)
	if err != nil {
		return nil, fmt.Errorf("load existing purchase order item receipts: %w", err)
	}
	for rows.Next() {
		var (
			productID   string
			receivedQty float64
		)
		if scanErr := rows.Scan(&productID, &receivedQty); scanErr != nil {
			rows.Close()
			return nil, fmt.Errorf("scan purchase order item receipts: %w", scanErr)
		}
		existingItemReceipts[productID] = receivedQty
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate purchase order item receipts: %w", err)
	}
	rows.Close()

	if _, err := tx.Exec(ctx, `
		UPDATE purchase_order_items
		SET deleted = TRUE,
		    deleted_at = CURRENT_TIMESTAMP,
		    deleted_by = NULLIF($3, '')::uuid
		WHERE business_id = $1
		  AND purchase_order_id = $2::uuid
		  AND deleted_at IS NULL
	`, req.BusinessID, req.PurchaseOrderID, req.UpdatedBy); err != nil {
		return nil, fmt.Errorf("soft delete purchase order items: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE purchase_order_additional_expenses
		SET deleted = TRUE,
		    deleted_at = CURRENT_TIMESTAMP,
		    deleted_by = NULLIF($3, '')::uuid
		WHERE business_id = $1
		  AND purchase_order_id = $2::uuid
		  AND deleted_at IS NULL
	`, req.BusinessID, req.PurchaseOrderID, req.UpdatedBy); err != nil {
		return nil, fmt.Errorf("soft delete purchase order additional expenses: %w", err)
	}

	for idx, item := range req.Items {
		var productName, sku, unitName string
		if err := tx.QueryRow(ctx, `
			SELECT
				p.name,
				COALESCE(p.sku, ''),
				COALESCE(u.name, '')
			FROM products p
			LEFT JOIN business_units u ON u.id = p.unit_id
			WHERE p.business_id = $1
			  AND p.id = $2::uuid
			  AND p.deleted_at IS NULL
			LIMIT 1
		`, req.BusinessID, item.ProductID).Scan(&productName, &sku, &unitName); err != nil {
			if err == pgx.ErrNoRows {
				return nil, fmt.Errorf("purchase order item product not found")
			}
			return nil, fmt.Errorf("resolve purchase order item product: %w", err)
		}

		if strings.TrimSpace(productName) == "" {
			productName = strings.TrimSpace(item.ProductID)
		}
		if strings.TrimSpace(sku) == "" {
			sku = ""
		}
		if strings.TrimSpace(unitName) == "" {
			unitName = ""
		}

		receivedQuantity := 0.0
		if item.ReceivedQuantity != nil {
			receivedQuantity = *item.ReceivedQuantity
		} else if previousReceived, ok := existingItemReceipts[item.ProductID]; ok {
			receivedQuantity = previousReceived
		}
		if receivedQuantity < 0 {
			receivedQuantity = 0
		}
		if receivedQuantity > item.OrderQuantity {
			receivedQuantity = item.OrderQuantity
		}
		balanceQuantity := item.OrderQuantity - receivedQuantity
		receivedStatus := "pending"
		if receivedQuantity > 0 && balanceQuantity > 0 {
			receivedStatus = "partial"
		} else if balanceQuantity <= 0 {
			receivedStatus = "received"
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO purchase_order_items (
				purchase_order_id,
				business_id,
				product_id,
				product_name,
				sku,
				unit,
				order_quantity,
				unit_cost_before_discount,
				discount_percentage,
				discount_amount,
				unit_cost_before_tax,
				product_tax_rate,
				tax_amount,
				net_cost,
				selling_price,
				line_cost,
				manufacture_date,
				expiry_date,
				lot_number,
				balance_quantity,
				received_quantity,
				items_received,
				received_status,
				sort_order
			) VALUES (
				NULLIF($1, '')::uuid,
				$2::uuid,
				$3::uuid,
				$4,
				$5,
				$6,
				$7,
				$8,
				$9,
				$10,
				$11,
				$12,
				$13,
				$14,
				$15,
				$16,
				NULLIF($17, '')::date,
				NULLIF($18, '')::date,
				$19,
				$21,
				$20,
				$20,
				$22,
				$23
			)
		`, req.PurchaseOrderID, req.BusinessID, item.ProductID, productName, sku, unitName, item.OrderQuantity, item.UnitCostBeforeDiscount, item.DiscountPercentage, item.DiscountAmount, item.UnitCostBeforeTax, item.ProductTaxRate, item.TaxAmount, item.NetCost, item.SellingPrice, item.LineCost, item.ManufactureDate, item.ExpiryDate, item.LotNumber, receivedQuantity, balanceQuantity, receivedStatus, idx); err != nil {
			return nil, fmt.Errorf("insert purchase order item: %w", err)
		}
	}

	for _, expense := range req.AdditionalExpenses {
		if _, err := tx.Exec(ctx, `
			INSERT INTO purchase_order_additional_expenses (
				purchase_order_id,
				business_id,
				expense_name,
				amount,
				sort_order
			) VALUES (
				NULLIF($1, '')::uuid,
				$2::uuid,
				$3,
				$4,
				$5
			)
		`, req.PurchaseOrderID, req.BusinessID, expense.Name, expense.Amount, expense.SortOrder); err != nil {
			return nil, fmt.Errorf("insert purchase order additional expense: %w", err)
		}
	}

	updated, err := getPurchaseOrderByID(ctx, tx, req.BusinessID, req.PurchaseOrderID)
	if err != nil {
		return nil, err
	}

	if err := syncPurchaseOrderInventoryTx(ctx, tx, req, existing, existingItemReceipts, req.Items); err != nil {
		return nil, err
	}

	if strings.TrimSpace(req.ActivityAction) != "" && strings.TrimSpace(req.ActivityNote) != "" {
		if err := insertPurchaseOrderLog(ctx, tx, CreatePurchaseOrderLogInput{
			BusinessID:      req.BusinessID,
			PurchaseOrderID: req.PurchaseOrderID,
			Action:          req.ActivityAction,
			ActionedBy:      req.ActivityActionedBy,
			Note:            req.ActivityNote,
		}); err != nil {
			return nil, err
		}
	}

	if strings.EqualFold(strings.TrimSpace(req.Status), "pending_approval") {
		if err := insertPurchaseOrderApproval(ctx, tx, CreatePurchaseOrderApprovalInput{
			BusinessID:       req.BusinessID,
			PurchaseOrderID:  req.PurchaseOrderID,
			ApprovalStatus:   "pending_approval",
			ReminderChannels: req.ApprovalReminderChannels,
			ReminderMessage:  req.ApprovalReminderMessage,
			Note:             req.ActivityNote,
			RequestedBy:      req.UpdatedBy,
		}); err != nil {
			return nil, err
		}

		if err := insertPurchaseOrderNotification(ctx, tx, CreatePurchaseOrderNotificationInput{
			BusinessID:              req.BusinessID,
			PurchaseOrderID:         req.PurchaseOrderID,
			PurchaseOrderStatusCode: "pending_approval",
			Channels:                req.ApprovalReminderChannels,
			Receivers:               req.ApprovalReminderReceivers,
			Message:                 req.ApprovalReminderMessage,
			Note:                    req.ActivityNote,
			CreatedBy:               req.UpdatedBy,
		}); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit purchase order update tx: %w", err)
	}

	return updated, nil
}

func DeletePurchaseOrderRepository(pool *pgxpool.Pool, businessID, purchaseOrderID, actionedBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	purchaseOrderID = strings.TrimSpace(purchaseOrderID)
	if businessID == "" || purchaseOrderID == "" {
		return ErrBusinessNotResolved
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin purchase order delete tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var currentStatus string
	if err := tx.QueryRow(ctx, `
		SELECT status
		FROM purchase_orders
		WHERE business_id = $1
		  AND id = $2::uuid
		  AND deleted_at IS NULL
		LIMIT 1
	`, businessID, purchaseOrderID).Scan(&currentStatus); err != nil {
		if err == pgx.ErrNoRows {
			return ErrPurchaseOrderNotFound
		}
		return fmt.Errorf("load purchase order for delete: %w", err)
	}

	statusDefinition, err := getPurchaseOrderStatusByCode(ctx, tx, currentStatus)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("purchase order status definition not found")
		}
		return fmt.Errorf("load purchase order status definition: %w", err)
	}
	if statusDefinition != nil && !statusDefinition.CanBeDeleted {
		return ErrPurchaseOrderCannotDelete
	}

	commandTag, err := tx.Exec(ctx, `
		UPDATE purchase_orders
		SET deleted = TRUE,
		    deleted_at = CURRENT_TIMESTAMP
		WHERE business_id = $1
		  AND id = $2::uuid
		  AND deleted_at IS NULL
	`, businessID, purchaseOrderID)
	if err != nil {
		return fmt.Errorf("soft delete purchase order: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrPurchaseOrderNotFound
	}

	if _, err := tx.Exec(ctx, `
		UPDATE purchase_order_items
		SET deleted = TRUE,
		    deleted_at = CURRENT_TIMESTAMP
		WHERE purchase_order_id = $1::uuid
		  AND deleted_at IS NULL
	`, purchaseOrderID); err != nil {
		return fmt.Errorf("soft delete purchase order items: %w", err)
	}

	if err := insertPurchaseOrderLog(ctx, tx, CreatePurchaseOrderLogInput{
		BusinessID:      businessID,
		PurchaseOrderID: purchaseOrderID,
		Action:          "deleted",
		ActionedBy:      actionedBy,
		Note:            "Purchase order deleted.",
	}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit purchase order delete tx: %w", err)
	}

	return nil
}

func getPurchaseOrderByID(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID, purchaseOrderID string) (*PurchaseOrder, error) {
	row := querier.QueryRow(ctx, purchaseOrderSelectQuery()+`
		WHERE po.business_id = $1
		  AND po.id = $2::uuid
		  AND po.deleted_at IS NULL
		LIMIT 1
	`, businessID, purchaseOrderID)

	order, err := scanPurchaseOrder(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPurchaseOrderNotFound
		}
		return nil, err
	}

	return &order, nil
}

func purchaseOrderSelectQuery() string {
	return `
		SELECT
			po.id::text,
			po.business_id::text,
			po.supplier_id::text,
			COALESCE(
				NULLIF(bs.business_name, ''),
				NULLIF(TRIM(CONCAT_WS(' ', bs.prefix, bs.first_name, bs.middle_name, bs.last_name)), ''),
				bs.contact_id
			) AS supplier_name,
			po.location_id::text,
			loc.location_name,
			po.reference_number,
			po.order_date::text,
			COALESCE(po.delivery_date::text, '') AS delivery_date,
			po.payment_term_value,
			po.payment_term_unit,
			COALESCE(po.attachment_name, '') AS attachment_name,
			COALESCE(po.attachment_url, '') AS attachment_url,
			COALESCE(po.delivery_address, '') AS delivery_address,
			po.delivery_charges,
			COALESCE(po.delivery_document_name, '') AS delivery_document_name,
			COALESCE(po.delivery_document_url, '') AS delivery_document_url,
			po.order_discount_amount,
			COALESCE(po.notes, '') AS notes,
			po.status,
			po.delivery_status,
			po.payment_status,
			po.subtotal,
			po.total_discount,
			po.total_tax,
			po.grand_total,
			po.items_count,
			po.total_quantity,
			COALESCE(u.id::text, '') AS created_by_id,
			COALESCE(u.full_name, 'System') AS created_by_name,
			po.created_at::text,
			po.updated_at::text
		FROM purchase_orders po
		INNER JOIN business_suppliers bs ON bs.id = po.supplier_id
		INNER JOIN business_locations loc ON loc.id = po.location_id
		LEFT JOIN users u ON u.id = po.created_by`
}

func purchaseOrderListQuery(businessID string, filters ListPurchaseOrdersFilters) (string, []any) {
	query := purchaseOrderSelectQuery()
	conditions := []string{
		"po.business_id = $1",
		"po.deleted_at IS NULL",
	}
	args := []any{businessID}

	addCondition := func(condition string, values ...any) {
		updatedCondition := condition
		for _, value := range values {
			args = append(args, value)
			placeholder := fmt.Sprintf("$%d", len(args))
			updatedCondition = strings.Replace(updatedCondition, "?", placeholder, 1)
		}
		conditions = append(conditions, updatedCondition)
	}

	if value := strings.TrimSpace(filters.LocationID); value != "" {
		addCondition("po.location_id::text = ?", value)
	}
	if value := strings.TrimSpace(filters.SupplierID); value != "" {
		addCondition("po.supplier_id::text = ?", value)
	}
	if value := strings.TrimSpace(filters.Status); value != "" {
		addCondition("po.status = ?", strings.ToLower(value))
	}
	if value := strings.TrimSpace(filters.DeliveryStatus); value != "" {
		addCondition("po.delivery_status = ?", strings.ToLower(value))
	}
	if value := strings.TrimSpace(filters.PaymentStatus); value != "" {
		addCondition("po.payment_status = ?", strings.ToLower(value))
	}
	if value := strings.TrimSpace(filters.DateFrom); value != "" {
		addCondition("po.order_date::date >= ?::date", value)
	}
	if value := strings.TrimSpace(filters.DateTo); value != "" {
		addCondition("po.order_date::date <= ?::date", value)
	}
	if value := strings.TrimSpace(filters.SearchQuery); value != "" {
		search := "%" + strings.ToLower(value) + "%"
		addCondition(`(
			LOWER(po.reference_number) LIKE ?
			OR LOWER(COALESCE(bs.business_name, '')) LIKE ?
			OR LOWER(COALESCE(TRIM(CONCAT_WS(' ', bs.prefix, bs.first_name, bs.middle_name, bs.last_name)), '')) LIKE ?
			OR LOWER(COALESCE(loc.location_name, '')) LIKE ?
		)`, search, search, search, search)
	}

	if len(conditions) > 0 {
		query += "\n\t\tWHERE " + strings.Join(conditions, "\n\t\t  AND ")
	}
	query += "\n\t\tORDER BY po.created_at DESC"
	return query, args
}

func scanPurchaseOrder(scanner interface {
	Scan(dest ...any) error
}) (PurchaseOrder, error) {
	var (
		order          PurchaseOrder
		deliveryDate   sql.NullString
		createdByID    sql.NullString
		createdByName  sql.NullString
		attachmentName sql.NullString
		attachmentURL  sql.NullString
		notes          sql.NullString
	)

	if err := scanner.Scan(
		&order.ID,
		&order.BusinessID,
		&order.SupplierID,
		&order.SupplierName,
		&order.LocationID,
		&order.LocationName,
		&order.ReferenceNumber,
		&order.OrderDate,
		&deliveryDate,
		&order.PaymentTermValue,
		&order.PaymentTermUnit,
		&attachmentName,
		&attachmentURL,
		&order.DeliveryAddress,
		&order.DeliveryCharges,
		&order.DeliveryDocumentName,
		&order.DeliveryDocumentURL,
		&order.OrderDiscountAmount,
		&notes,
		&order.Status,
		&order.DeliveryStatus,
		&order.PaymentStatus,
		&order.Subtotal,
		&order.TotalDiscount,
		&order.TotalTax,
		&order.GrandTotal,
		&order.ItemsCount,
		&order.TotalQuantity,
		&createdByID,
		&createdByName,
		&order.CreatedAt,
		&order.UpdatedAt,
	); err != nil {
		return PurchaseOrder{}, err
	}

	order.DeliveryDate = ""
	if deliveryDate.Valid {
		order.DeliveryDate = deliveryDate.String
	}
	order.AttachmentName = attachmentName.String
	order.AttachmentURL = attachmentURL.String
	order.Notes = notes.String

	if createdByID.Valid || createdByName.Valid {
		order.CreatedBy = &PurchaseOrderCreatedBy{
			ID:   createdByID.String,
			Name: createdByName.String,
		}
	}

	return order, nil
}
