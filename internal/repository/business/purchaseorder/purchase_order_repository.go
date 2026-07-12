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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	rows, err := pool.Query(ctx, purchaseOrderSelectQuery()+`
		WHERE po.business_id = $1
		  AND po.deleted_at IS NULL
		ORDER BY po.created_at DESC
	`, businessID)
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
			$11,
			$12,
			$13,
			$14,
			$15,
			$16,
			$17,
			$18,
			$19,
			$20,
			NULLIF($21, '')::uuid
		)
		RETURNING id::text
	`, req.BusinessID, req.SupplierID, req.LocationID, req.ReferenceNumber, req.OrderDate, req.DeliveryDate, req.PaymentTermValue, req.PaymentTermUnit, req.AttachmentName, req.AttachmentURL, req.Notes, req.Status, req.DeliveryStatus, req.PaymentStatus, req.Subtotal, req.TotalDiscount, req.TotalTax, req.GrandTotal, req.ItemsCount, req.TotalQuantity, req.CreatedBy)

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
				expiry_date,
				lot_number,
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
				$18,
				0,
				'pending',
				$19
			)
		`, item.PurchaseOrderID, req.BusinessID, item.ProductID, productName, sku, unitName, item.OrderQuantity, item.UnitCostBeforeDiscount, item.DiscountPercentage, item.DiscountAmount, item.UnitCostBeforeTax, item.ProductTaxRate, item.TaxAmount, item.NetCost, item.SellingPrice, item.LineCost, item.ExpiryDate, item.LotNumber, idx); err != nil {
			return nil, fmt.Errorf("insert purchase order item: %w", err)
		}
	}

	created, err := getPurchaseOrderByID(ctx, tx, req.BusinessID, purchaseOrderID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit purchase order tx: %w", err)
	}

	return created, nil
}

func DeletePurchaseOrderRepository(pool *pgxpool.Pool, businessID, purchaseOrderID string) error {
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
