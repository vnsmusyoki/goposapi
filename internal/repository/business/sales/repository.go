package sales

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func CreateSaleOrderRepository(pool *pgxpool.Pool, req CreateSaleOrderInput) (*Sale, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.LocationID = strings.TrimSpace(req.LocationID)
	req.CustomerID = strings.TrimSpace(req.CustomerID)
	req.ReferenceNumber = strings.TrimSpace(req.ReferenceNumber)
	req.SaleDate = strings.TrimSpace(req.SaleDate)
	req.CustomerName = strings.TrimSpace(req.CustomerName)
	req.CustomerPhone = strings.TrimSpace(req.CustomerPhone)
	req.CustomerEmail = strings.ToLower(strings.TrimSpace(req.CustomerEmail))
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	req.Notes = strings.TrimSpace(req.Notes)
	req.StockAccountingMethod = strings.TrimSpace(req.StockAccountingMethod)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)

	if req.BusinessID == "" || req.LocationID == "" || req.SaleDate == "" || len(req.Items) == 0 {
		return nil, ErrInvalidSaleInput
	}

	if req.Status == "" {
		req.Status = "draft"
	}
	if req.StockAccountingMethod == "" {
		req.StockAccountingMethod = "FIFO"
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin sale tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var saleID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO sales_orders (
			business_id,
			location_id,
			customer_id,
			reference_number,
			sale_date,
			customer_name,
			customer_phone,
			customer_email,
			status,
			subtotal,
			total_discount,
			total_tax,
			grand_total,
			items_count,
			total_quantity,
			notes,
			stock_accounting_method,
			reserve_order_items,
			created_by
		) VALUES (
			$1::uuid,
			$2::uuid,
			NULLIF($3, '')::uuid,
			$4,
			$5::timestamptz,
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
			$17,
			$18,
			NULLIF($19, '')::uuid
		)
		RETURNING id::text
	`, req.BusinessID, req.LocationID, req.CustomerID, req.ReferenceNumber, req.SaleDate, req.CustomerName, req.CustomerPhone, req.CustomerEmail, req.Status, req.Subtotal, req.TotalDiscount, req.TotalTax, req.GrandTotal, req.ItemsCount, req.TotalQuantity, req.Notes, req.StockAccountingMethod, req.ReserveOrderItems, req.CreatedBy).Scan(&saleID); err != nil {
		return nil, fmt.Errorf("insert sale: %w", err)
	}

	for idx, item := range req.Items {
		productName, sku, unitName, err := loadSaleProductSnapshotTx(ctx, tx, req.BusinessID, item.ProductID)
		if err != nil {
			return nil, err
		}

		batchTrackingEnabled := true
		var saleItemID string
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
		`, saleID, req.BusinessID, item.ProductID, productName, sku, unitName, item.Quantity, item.UnitCost, item.DiscountPercentage, item.DiscountAmount, item.TaxRate, item.TaxAmount, item.UnitPrice, item.LineTotal, batchTrackingEnabled, idx).Scan(&saleItemID); err != nil {
			return nil, fmt.Errorf("insert sale item: %w", err)
		}

		if err := applySalesOrderInventoryAllocationTx(ctx, tx, req, item, saleID, saleItemID, productName, sku, unitName, req.Status == "approved" && req.ReserveOrderItems); err != nil {
			return nil, err
		}
	}

	created, err := GetSaleByIDRepositoryTx(ctx, tx, req.BusinessID, saleID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit sale tx: %w", err)
	}

	return created, nil
}

func GetSaleByIDRepositoryTx(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID, saleID string) (*Sale, error) {
	row := querier.QueryRow(ctx, `
		SELECT
			id::text,
			business_id::text,
			location_id::text,
			COALESCE(customer_id::text, ''),
			reference_number,
			sale_date::text,
			COALESCE(customer_name, ''),
			COALESCE(customer_phone, ''),
			COALESCE(customer_email, ''),
			status,
			subtotal,
			total_discount,
			total_tax,
			grand_total,
			items_count,
			total_quantity,
			COALESCE(notes, ''),
			COALESCE(stock_accounting_method, ''),
			COALESCE(reserve_order_items, FALSE),
			COALESCE(sale_id::text, ''),
			COALESCE(converted_at::text, ''),
			COALESCE(created_by::text, ''),
			created_at::text,
			updated_at::text
		FROM sales_orders
		WHERE business_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
		LIMIT 1
	`, businessID, saleID)

	var sale Sale
	if err := row.Scan(
		&sale.ID,
		&sale.BusinessID,
		&sale.LocationID,
		&sale.CustomerID,
		&sale.ReferenceNumber,
		&sale.SaleDate,
		&sale.CustomerName,
		&sale.CustomerPhone,
		&sale.CustomerEmail,
		&sale.Status,
		&sale.Subtotal,
		&sale.TotalDiscount,
		&sale.TotalTax,
		&sale.GrandTotal,
		&sale.ItemsCount,
		&sale.TotalQuantity,
		&sale.Notes,
		&sale.StockAccountingMethod,
		&sale.ReserveOrderItems,
		&sale.SaleID,
		&sale.ConvertedAt,
		&sale.CreatedBy,
		&sale.CreatedAt,
		&sale.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSaleNotFound
		}
		return nil, fmt.Errorf("load sale: %w", err)
	}

	return &sale, nil
}

func loadSaleProductSnapshotTx(ctx context.Context, tx saleInventoryTx, businessID, productID string) (string, string, string, error) {
	var productName, sku, unitName sql.NullString
	if err := tx.QueryRow(ctx, `
		SELECT
			p.name,
			COALESCE(p.sku, ''),
			COALESCE(u.name, '')
		FROM products p
		LEFT JOIN business_units u ON u.id = p.unit_id
		WHERE p.business_id = $1::uuid
		  AND p.id = $2::uuid
		  AND p.deleted_at IS NULL
		LIMIT 1
	`, businessID, productID).Scan(&productName, &sku, &unitName); err != nil {
		if err == pgx.ErrNoRows {
			return "", "", "", fmt.Errorf("sale product not found")
		}
		return "", "", "", fmt.Errorf("load sale product: %w", err)
	}

	return productName.String, sku.String, unitName.String, nil
}

func saleStatusReservesInventory(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "approved", "processing":
		return true
	default:
		return false
	}
}

func saleStatusConsumesInventory(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ready_for_shipment", "completed":
		return true
	default:
		return false
	}
}
