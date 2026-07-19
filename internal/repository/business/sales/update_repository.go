package sales

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type salesOrderUpdateSnapshot struct {
	ID                    string
	ReferenceNumber       string
	LocationID            string
	Status                string
	StockAccountingMethod string
	ReserveOrderItems     bool
	SaleID                string
}

func UpdateSaleOrderRepository(pool *pgxpool.Pool, req UpdateSaleOrderInput) (*Sale, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.SalesOrderID = strings.TrimSpace(req.SalesOrderID)
	req.CustomerID = strings.TrimSpace(req.CustomerID)
	req.LocationID = strings.TrimSpace(req.LocationID)
	req.SaleDate = strings.TrimSpace(req.SaleDate)
	req.CustomerName = strings.TrimSpace(req.CustomerName)
	req.CustomerPhone = strings.TrimSpace(req.CustomerPhone)
	req.CustomerEmail = strings.ToLower(strings.TrimSpace(req.CustomerEmail))
	req.Notes = strings.TrimSpace(req.Notes)
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	req.UpdatedBy = strings.TrimSpace(req.UpdatedBy)
	req.UpdatedByName = strings.TrimSpace(req.UpdatedByName)

	if req.BusinessID == "" || req.SalesOrderID == "" || req.LocationID == "" || req.SaleDate == "" || len(req.Items) == 0 {
		return nil, ErrInvalidSaleInput
	}
	if req.Status == "" {
		req.Status = "draft"
	}
	if !isEditableSaleOrderStatus(req.Status) {
		return nil, ErrSalesOrderCannotUpdate
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin sale order update tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err := loadSalesOrderUpdateSnapshotTx(ctx, tx, req.BusinessID, req.SalesOrderID)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(current.SaleID) != "" || saleStatusConsumesInventory(current.Status) {
		return nil, ErrSalesOrderCannotUpdate
	}

	if current.ReserveOrderItems || current.Status == "approved" || current.Status == "processing" {
		if err := releaseSalesOrderReservationTx(ctx, tx, req.BusinessID, req.SalesOrderID); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM sales_order_item_batch_allocations
		WHERE sales_order_id = $1::uuid
	`, req.SalesOrderID); err != nil {
		return nil, fmt.Errorf("delete sales order batch allocations: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE sales_order_items
		SET deleted = TRUE,
		    deleted_at = CURRENT_TIMESTAMP,
		    deleted_by = NULLIF($2, '')::uuid
		WHERE sales_order_id = $1::uuid
		  AND deleted_at IS NULL
	`, req.SalesOrderID, req.UpdatedBy); err != nil {
		return nil, fmt.Errorf("soft delete sales order items: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE sales_orders
		SET customer_id = NULLIF($3, '')::uuid,
		    location_id = $4::uuid,
		    sale_date = $5::timestamptz,
		    customer_name = $6,
		    customer_phone = $7,
		    customer_email = $8,
		    status = $9,
		    subtotal = $10,
		    total_discount = $11,
		    total_tax = $12,
		    grand_total = $13,
		    items_count = $14,
		    total_quantity = $15,
		    notes = $16,
		    reserve_order_items = $17,
		    updated_at = CURRENT_TIMESTAMP
		WHERE business_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
	`, req.BusinessID, req.SalesOrderID, req.CustomerID, req.LocationID, req.SaleDate, req.CustomerName, req.CustomerPhone, req.CustomerEmail, req.Status, req.Subtotal, req.TotalDiscount, req.TotalTax, req.GrandTotal, req.ItemsCount, req.TotalQuantity, req.Notes, req.ReserveOrderItems); err != nil {
		return nil, fmt.Errorf("update sale order: %w", err)
	}

	for idx, item := range req.Items {
		productName, sku, unitName, err := loadSaleProductSnapshotTx(ctx, tx, req.BusinessID, item.ProductID)
		if err != nil {
			return nil, err
		}

		batchTrackingEnabled := true
		var saleOrderItemID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO sales_order_items (
				sales_order_id,
				business_id,
				product_id,
				product_name,
				sku,
				unit,
				quantity,
				unit_cost,
				discount_percentage,
				discount_amount,
				tax_rate,
				tax_amount,
				unit_price,
				line_total,
				batch_tracking_enabled,
				sort_order
			)
			VALUES (
				$1::uuid,
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
				$16
			)
			RETURNING id::text
		`, req.SalesOrderID, req.BusinessID, item.ProductID, productName, sku, unitName, item.Quantity, item.UnitCost, item.DiscountPercentage, item.DiscountAmount, item.TaxRate, item.TaxAmount, item.UnitPrice, item.LineTotal, batchTrackingEnabled, idx).Scan(&saleOrderItemID); err != nil {
			return nil, fmt.Errorf("insert sales order item: %w", err)
		}

		if err := applySalesOrderInventoryAllocationTx(ctx, tx, CreateSaleOrderInput{
			BusinessID:            req.BusinessID,
			LocationID:            req.LocationID,
			StockAccountingMethod: current.StockAccountingMethod,
		}, item, req.SalesOrderID, saleOrderItemID, productName, sku, unitName, req.ReserveOrderItems && (req.Status == "approved" || req.Status == "processing")); err != nil {
			return nil, err
		}
	}

	if err := CreateSalesOrderLogTx(ctx, tx, SalesOrderLogInput{
		BusinessID:   req.BusinessID,
		SalesOrderID: req.SalesOrderID,
		Action:       "updated",
		ActionedBy:   req.UpdatedBy,
		Note:         buildSalesOrderActivityNote("updated", current.ReferenceNumber, req.UpdatedByName, current.Status, req.Status, req.ReserveOrderItems, false),
	}); err != nil {
		return nil, err
	}

	updated, err := GetSaleByIDRepositoryTx(ctx, tx, req.BusinessID, req.SalesOrderID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit sale order update tx: %w", err)
	}

	return updated, nil
}

func loadSalesOrderUpdateSnapshotTx(ctx context.Context, tx saleInventoryTx, businessID, salesOrderID string) (*salesOrderUpdateSnapshot, error) {
	var snapshot salesOrderUpdateSnapshot
	if err := tx.QueryRow(ctx, `
		SELECT
			id::text,
			reference_number,
			location_id::text,
			status,
			COALESCE(stock_accounting_method, 'FIFO'),
			COALESCE(reserve_order_items, FALSE),
			COALESCE(sale_id::text, '')
		FROM sales_orders
		WHERE business_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
		LIMIT 1
	`, businessID, salesOrderID).Scan(&snapshot.ID, &snapshot.ReferenceNumber, &snapshot.LocationID, &snapshot.Status, &snapshot.StockAccountingMethod, &snapshot.ReserveOrderItems, &snapshot.SaleID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSaleNotFound
		}
		return nil, fmt.Errorf("load sales order update snapshot: %w", err)
	}

	return &snapshot, nil
}

func isEditableSaleOrderStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "draft", "pending_approval", "approved", "processing":
		return true
	default:
		return false
	}
}
