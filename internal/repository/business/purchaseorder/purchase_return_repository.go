package purchaseorder

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PurchaseReturnableStockItem struct {
	ID                     string  `json:"id"`
	ProductID              string  `json:"productId"`
	ProductName            string  `json:"productName"`
	SKU                    string  `json:"sku"`
	SupplierID             string  `json:"supplierId"`
	SupplierName           string  `json:"supplierName"`
	LocationID             string  `json:"locationId"`
	LocationName           string  `json:"locationName"`
	LotNumber              string  `json:"lotNumber"`
	BatchNumber            string  `json:"batchNumber"`
	ExpiryDate             *string `json:"expiryDate"`
	SuppliedBySupplier     float64 `json:"suppliedBySupplier"`
	SoldAlreadyForSupplier float64 `json:"soldAlreadyForSupplier"`
	AvailableQuantity      float64 `json:"availableQuantity"`
	UnitPrice              float64 `json:"unitPrice"`
	UnitCostBeforeTax      float64 `json:"unitCostBeforeTax"`
	CurrentStock           float64 `json:"currentStock"`
	ReceivedAt             string  `json:"receivedAt"`
	SourceReference        string  `json:"sourceReference"`
	SourceID               string  `json:"sourceId"`
}

func SearchPurchaseReturnableStockRepository(pool *pgxpool.Pool, businessID, query, locationID, supplierID string) ([]PurchaseReturnableStockGroup, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	query = strings.TrimSpace(query)
	locationID = strings.TrimSpace(locationID)
	supplierID = strings.TrimSpace(supplierID)

	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}
	if query == "" {
		return []PurchaseReturnableStockGroup{}, nil
	}

	args := []any{businessID, "%" + strings.ToLower(query) + "%"}
	where := `
		WHERE ib.business_id = $1::uuid
		  AND ib.quantity_remaining > 0
		  AND ib.source_type = 'purchase_order'
		  AND (
			LOWER(COALESCE(p.name, '')) LIKE $2
			OR LOWER(COALESCE(p.sku, '')) LIKE $2
			OR LOWER(COALESCE(ib.lot_number, '')) LIKE $2
			OR LOWER(COALESCE(ib.batch_number, '')) LIKE $2
			OR LOWER(COALESCE(po.reference_number, '')) LIKE $2
		  )
	`

	if locationID != "" && locationID != "all" {
		args = append(args, locationID)
		where += fmt.Sprintf(" AND ib.location_id = $%d::uuid", len(args))
	}

	if supplierID != "" && supplierID != "all" {
		args = append(args, supplierID)
		where += fmt.Sprintf(" AND COALESCE(ib.supplier_id, po.supplier_id) = $%d::uuid", len(args))
	}

	rows, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT
			ib.id::text,
			p.id::text,
			p.name,
			COALESCE(p.sku, ''),
			COALESCE(ib.supplier_id::text, po.supplier_id::text, ''),
			COALESCE(
				NULLIF(bs.business_name, ''),
				NULLIF(TRIM(CONCAT_WS(' ', bs.prefix, bs.first_name, bs.middle_name, bs.last_name)), ''),
				COALESCE(bs.contact_id, '')
			),
			ib.location_id::text,
			COALESCE(loc.location_name, ''),
			COALESCE(ib.lot_number, ''),
			COALESCE(ib.batch_number, ''),
			COALESCE(ib.expiry_date::text, ''),
			COALESCE(ib.quantity_received, 0),
			COALESCE(GREATEST(ib.quantity_received - ib.quantity_remaining, 0), 0),
			COALESCE(ib.quantity_remaining, 0),
			COALESCE(ib.unit_cost, 0),
			COALESCE(ib.unit_cost, 0),
			COALESCE(balance.quantity_available, 0),
			COALESCE(ib.received_at::text, ''),
			COALESCE(po.reference_number, ''),
			COALESCE(ib.source_id::text, '')
		FROM inventory_batches ib
		JOIN products p ON p.id = ib.product_id
		LEFT JOIN purchase_orders po ON po.id = ib.source_id::uuid AND ib.source_type = 'purchase_order'
		LEFT JOIN business_suppliers bs ON bs.id = COALESCE(ib.supplier_id, po.supplier_id) AND bs.business_id = ib.business_id AND bs.deleted_at IS NULL
		LEFT JOIN business_locations loc ON loc.id = ib.location_id
		LEFT JOIN LATERAL (
			SELECT COALESCE(SUM(quantity_available), 0) AS quantity_available
			FROM inventory_balances
			WHERE business_id = ib.business_id
			  AND product_id = ib.product_id
			  AND location_id = ib.location_id
		) balance ON TRUE
		%s
		ORDER BY COALESCE(ib.expiry_date, DATE '9999-12-31') ASC, ib.received_at ASC, p.name ASC
	`, where), args...)
	if err != nil {
		return nil, fmt.Errorf("search purchase returnable stock: %w", err)
	}
	defer rows.Close()

	items := make([]PurchaseReturnableStockItem, 0)
	for rows.Next() {
		var item PurchaseReturnableStockItem
		var expiry sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.ProductID,
			&item.ProductName,
			&item.SKU,
			&item.SupplierID,
			&item.SupplierName,
			&item.LocationID,
			&item.LocationName,
			&item.LotNumber,
			&item.BatchNumber,
			&expiry,
			&item.SuppliedBySupplier,
			&item.SoldAlreadyForSupplier,
			&item.AvailableQuantity,
			&item.UnitPrice,
			&item.UnitCostBeforeTax,
			&item.CurrentStock,
			&item.ReceivedAt,
			&item.SourceReference,
			&item.SourceID,
		); err != nil {
			return nil, fmt.Errorf("scan purchase returnable stock: %w", err)
		}
		if expiry.Valid {
			value := expiry.String
			item.ExpiryDate = &value
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purchase returnable stock: %w", err)
	}

	return groupPurchaseReturnableStockItems(items), nil
}

type purchaseReturnTx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func ListPurchaseReturnsRepository(pool *pgxpool.Pool, businessID string, filters ListPurchaseReturnsFilters) ([]PurchaseReturn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	query, args := purchaseReturnListQuery(businessID, filters)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list purchase returns: %w", err)
	}
	defer rows.Close()

	returns := make([]PurchaseReturn, 0)
	for rows.Next() {
		entry, err := scanPurchaseReturn(rows)
		if err != nil {
			return nil, err
		}
		returns = append(returns, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purchase returns: %w", err)
	}

	return returns, nil
}

func GetPurchaseReturnByIDRepository(pool *pgxpool.Pool, businessID, purchaseReturnID string) (*PurchaseReturn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	purchaseReturnID = strings.TrimSpace(purchaseReturnID)
	if businessID == "" || purchaseReturnID == "" {
		return nil, ErrBusinessNotResolved
	}

	return getPurchaseReturnByID(ctx, pool, businessID, purchaseReturnID)
}

func GetPurchaseReturnItemsRepository(pool *pgxpool.Pool, businessID, purchaseReturnID string) ([]PurchaseReturnItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	purchaseReturnID = strings.TrimSpace(purchaseReturnID)
	if businessID == "" || purchaseReturnID == "" {
		return nil, ErrBusinessNotResolved
	}

	return listPurchaseReturnItemsTx(ctx, pool, businessID, purchaseReturnID)
}

func CreatePurchaseReturnRepository(pool *pgxpool.Pool, req CreatePurchaseReturnInput) (*PurchaseReturn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.ParentPurchaseID = strings.TrimSpace(req.ParentPurchaseID)
	req.ParentPurchaseReference = strings.TrimSpace(req.ParentPurchaseReference)
	req.ReferenceNumber = strings.TrimSpace(req.ReferenceNumber)
	req.ReturnDate = strings.TrimSpace(req.ReturnDate)
	req.LocationID = strings.TrimSpace(req.LocationID)
	req.SupplierID = strings.TrimSpace(req.SupplierID)
	req.ReturnReason = strings.TrimSpace(req.ReturnReason)
	req.Notes = strings.TrimSpace(req.Notes)
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	req.PaymentStatus = strings.ToLower(strings.TrimSpace(req.PaymentStatus))
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)

	if req.BusinessID == "" || req.LocationID == "" {
		return nil, ErrInvalidPurchaseReturnInput
	}
	if len(req.Items) == 0 {
		return nil, ErrInvalidPurchaseReturnInput
	}
	if req.ReferenceNumber == "" {
		req.ReferenceNumber = fmt.Sprintf("PR-%s", time.Now().UTC().Format("20060102150405"))
	}
	if req.ReturnDate == "" {
		req.ReturnDate = time.Now().UTC().Format(time.RFC3339)
	}
	if req.Status == "" {
		req.Status = "returned"
	}
	if req.PaymentStatus == "" {
		req.PaymentStatus = "unpaid"
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin purchase return tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var returnID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO purchase_returns (
			business_id,
			parent_purchase_id,
			parent_purchase_reference,
			reference_number,
			return_date,
			location_id,
			supplier_id,
			status,
			payment_status,
			grand_total,
			payment_due,
			total_quantity,
			items_count,
			return_reason,
			notes,
			created_by,
			updated_by
		)
		VALUES (
			$1::uuid,
			NULLIF($2, '')::uuid,
			$3,
			$4,
			$5::timestamptz,
			$6::uuid,
			NULLIF($7, '')::uuid,
			$8,
			$9,
			0,
			0,
			0,
			0,
			$10,
			$11,
			NULLIF($12, '')::uuid,
			NULLIF($12, '')::uuid
		)
		RETURNING id::text
	`, req.BusinessID, req.ParentPurchaseID, req.ParentPurchaseReference, req.ReferenceNumber, req.ReturnDate, req.LocationID, req.SupplierID, req.Status, req.PaymentStatus, req.ReturnReason, req.Notes, req.CreatedBy).Scan(&returnID); err != nil {
		return nil, fmt.Errorf("insert purchase return header: %w", err)
	}

	summary, err := processPurchaseReturnItemsTx(ctx, tx, req, returnID)
	if err != nil {
		return nil, err
	}

	paymentDue := summary.GrandTotal
	if _, err := tx.Exec(ctx, `
		UPDATE purchase_returns
		SET parent_purchase_id = NULLIF($2, '')::uuid,
		    parent_purchase_reference = $3,
		    grand_total = $4,
		    payment_due = $5,
		    total_quantity = $6,
		    items_count = $7
		WHERE id = $1::uuid
	`, returnID, summary.ParentPurchaseID, summary.ParentPurchaseReference, summary.GrandTotal, paymentDue, summary.TotalQuantity, summary.ItemsCount); err != nil {
		return nil, fmt.Errorf("update purchase return totals: %w", err)
	}

	createdReturn, err := getPurchaseReturnByIDTx(ctx, tx, req.BusinessID, returnID)
	if err != nil {
		return nil, err
	}

	if err := insertPurchaseReturnLog(ctx, tx, CreatePurchaseReturnLogInput{
		BusinessID:       req.BusinessID,
		PurchaseReturnID: returnID,
		Action:           "created",
		ActionedBy:       req.CreatedBy,
		Note:             "Purchase return created.",
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit purchase return tx: %w", err)
	}

	return createdReturn, nil
}

type purchaseReturnProcessSummary struct {
	ParentPurchaseID        string
	ParentPurchaseReference string
	TotalQuantity           float64
	GrandTotal              float64
	ItemsCount              int
}

func processPurchaseReturnItemsTx(ctx context.Context, tx purchaseReturnTx, req CreatePurchaseReturnInput, returnID string) (purchaseReturnProcessSummary, error) {
	parentPurchaseIDs := make(map[string]struct{})
	summary := purchaseReturnProcessSummary{
		ParentPurchaseReference: strings.TrimSpace(req.ParentPurchaseReference),
	}

	for _, item := range req.Items {
		productID := strings.TrimSpace(item.ProductID)
		if productID == "" {
			continue
		}

		batchKey := strings.TrimSpace(item.BatchKey)
		supplierID := req.SupplierID
		if strings.Contains(batchKey, "::") {
			parts := strings.SplitN(batchKey, "::", 2)
			if len(parts) == 2 {
				if strings.TrimSpace(parts[0]) != "" {
					productID = strings.TrimSpace(parts[0])
				}
				if strings.TrimSpace(parts[1]) != "" {
					supplierID = strings.TrimSpace(parts[1])
				}
			}
		}

		quantity := item.Quantity
		if quantity <= 0 {
			return purchaseReturnProcessSummary{}, ErrInvalidPurchaseReturnInput
		}
		unitPrice := item.UnitPrice
		if unitPrice < 0 {
			return purchaseReturnProcessSummary{}, ErrInvalidPurchaseReturnInput
		}

		allocations, err := loadPurchaseReturnAllocationsTx(ctx, tx, req.BusinessID, productID, req.LocationID, supplierID)
		if err != nil {
			return purchaseReturnProcessSummary{}, err
		}

		available := 0.0
		for _, allocation := range allocations {
			available += allocation.QuantityRemaining
		}
		if available <= 0 {
			return purchaseReturnProcessSummary{}, fmt.Errorf("product %s is not stocked at the selected location", productID)
		}
		if quantity > available {
			return purchaseReturnProcessSummary{}, fmt.Errorf("purchase return quantity for %s exceeds available stock", productID)
		}

		remaining := quantity
		for _, allocation := range allocations {
			if remaining <= 0 {
				break
			}
			consume := allocation.QuantityRemaining
			if consume > remaining {
				consume = remaining
			}
			if consume <= 0 {
				continue
			}

			balanceSnap, err := getOrCreateInventoryBalanceTx(ctx, tx, req.BusinessID, productID, req.LocationID, -consume)
			if err != nil {
				return purchaseReturnProcessSummary{}, err
			}

			nextRemaining := allocation.QuantityRemaining - consume
			if _, err := tx.Exec(ctx, `
				UPDATE inventory_batches
				SET quantity_remaining = $2,
				    updated_at = CURRENT_TIMESTAMP
				WHERE id = $1::uuid
			`, allocation.BatchID, nextRemaining); err != nil {
				return purchaseReturnProcessSummary{}, fmt.Errorf("update inventory batch after purchase return: %w", err)
			}

			lineTotal := consume * unitPrice
			if _, err := tx.Exec(ctx, `
				INSERT INTO purchase_return_items (
					purchase_return_id,
					business_id,
					product_id,
					product_name,
					sku,
					supplier_id,
					supplier_name,
					location_id,
					location_name,
					purchase_order_id,
					inventory_batch_id,
					lot_number,
					batch_number,
					expiry_date,
					manufacture_date,
					quantity,
					unit_price,
					line_total
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					$3::uuid,
					$4,
					$5,
					NULLIF($6, '')::uuid,
					$7,
					$8::uuid,
					$9,
					NULLIF($10, '')::uuid,
					$11::uuid,
					$12,
					$13,
					NULLIF($14, '')::date,
					NULLIF($15, '')::date,
					$16,
					$17,
					$18
				)
			`, returnID, req.BusinessID, productID, allocation.ProductName, allocation.SKU, allocation.SupplierID, allocation.SupplierName, req.LocationID, allocation.LocationName, allocation.PurchaseOrderID, allocation.BatchID, allocation.LotNumber, allocation.BatchNumber, allocation.ExpiryDate, allocation.ManufactureDate, consume, unitPrice, lineTotal); err != nil {
				return purchaseReturnProcessSummary{}, fmt.Errorf("insert purchase return item: %w", err)
			}

			if err := insertStockMovementTx(ctx, tx, inventoryMovementInput{
				BusinessID:         req.BusinessID,
				PurchaseOrderID:    allocation.PurchaseOrderID,
				ReferenceNumber:    req.ReferenceNumber,
				LocationID:         req.LocationID,
				ProductID:          productID,
				InventoryBalanceID: balanceSnap.ID,
				InventoryBatchID:   allocation.BatchID,
				MovementType:       "purchase_return",
				QuantityIn:         0,
				QuantityOut:        consume,
				UnitCost:           unitPrice,
				StockBefore:        balanceSnap.QuantityAvailable,
				StockAfter:         balanceSnap.QuantityAvailable - consume,
				Note:               fmt.Sprintf("Returned stock for %s.", allocation.ProductName),
				PerformedBy:        req.CreatedBy,
			}); err != nil {
				return purchaseReturnProcessSummary{}, err
			}

			if allocation.PurchaseOrderID != "" {
				parentPurchaseIDs[allocation.PurchaseOrderID] = struct{}{}
				if summary.ParentPurchaseReference == "" && strings.TrimSpace(allocation.SourceReference) != "" {
					summary.ParentPurchaseReference = allocation.SourceReference
				}
			}

			remaining -= consume
			summary.TotalQuantity += consume
			summary.GrandTotal += lineTotal
			summary.ItemsCount++
		}

		if remaining > 0 {
			return purchaseReturnProcessSummary{}, fmt.Errorf("insufficient stock remaining for %s", productID)
		}
	}

	if len(parentPurchaseIDs) == 1 {
		for purchaseID := range parentPurchaseIDs {
			summary.ParentPurchaseID = purchaseID
		}
	} else {
		summary.ParentPurchaseID = ""
		if len(parentPurchaseIDs) > 1 {
			summary.ParentPurchaseReference = "Multiple"
		}
	}

	return summary, nil
}

func UpdatePurchaseReturnRepository(pool *pgxpool.Pool, req UpdatePurchaseReturnInput) (*PurchaseReturn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.PurchaseReturnID = strings.TrimSpace(req.PurchaseReturnID)
	req.LocationID = strings.TrimSpace(req.LocationID)
	req.SupplierID = strings.TrimSpace(req.SupplierID)
	req.ReturnReason = strings.TrimSpace(req.ReturnReason)
	req.Notes = strings.TrimSpace(req.Notes)
	req.UpdatedBy = strings.TrimSpace(req.UpdatedBy)

	if req.BusinessID == "" || req.PurchaseReturnID == "" || req.LocationID == "" || len(req.Items) == 0 {
		return nil, ErrInvalidPurchaseReturnInput
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin purchase return update tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	existing, err := getPurchaseReturnByIDTx(ctx, tx, req.BusinessID, req.PurchaseReturnID)
	if err != nil {
		return nil, err
	}

	for _, item := range existing.Items {
		if strings.TrimSpace(item.InventoryBatchID) == "" {
			continue
		}

		balanceSnap, err := getOrCreateInventoryBalanceTx(ctx, tx, req.BusinessID, item.ProductID, item.LocationID, item.Quantity)
		if err != nil {
			return nil, err
		}

		if _, err := tx.Exec(ctx, `
			UPDATE inventory_batches
			SET quantity_remaining = quantity_remaining + $2,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $1::uuid
		`, item.InventoryBatchID, item.Quantity); err != nil {
			return nil, fmt.Errorf("restore purchase return inventory batch: %w", err)
		}

		if err := insertStockMovementTx(ctx, tx, inventoryMovementInput{
			BusinessID:         req.BusinessID,
			PurchaseOrderID:    item.PurchaseOrderID,
			ReferenceNumber:    existing.ReferenceNumber,
			LocationID:         item.LocationID,
			ProductID:          item.ProductID,
			InventoryBalanceID: balanceSnap.ID,
			InventoryBatchID:   item.InventoryBatchID,
			MovementType:       "purchase_return_reversal",
			QuantityIn:         item.Quantity,
			QuantityOut:        0,
			UnitCost:           item.UnitPrice,
			StockBefore:        balanceSnap.QuantityAvailable,
			StockAfter:         balanceSnap.QuantityAvailable + item.Quantity,
			Note:               fmt.Sprintf("Reversed purchase return for %s.", item.ProductName),
			PerformedBy:        req.UpdatedBy,
		}); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE purchase_return_items
		SET deleted = TRUE,
		    deleted_at = CURRENT_TIMESTAMP
		WHERE purchase_return_id = $1::uuid
		  AND deleted_at IS NULL
	`, req.PurchaseReturnID); err != nil {
		return nil, fmt.Errorf("soft delete purchase return items: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE purchase_returns
		SET location_id = $3::uuid,
		    return_reason = $4,
		    notes = $5,
		    updated_by = NULLIF($6, '')::uuid,
		    updated_at = CURRENT_TIMESTAMP
		WHERE business_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
	`, req.BusinessID, req.PurchaseReturnID, req.LocationID, req.ReturnReason, req.Notes, req.UpdatedBy); err != nil {
		return nil, fmt.Errorf("update purchase return header: %w", err)
	}

	supplierID := strings.TrimSpace(existing.SupplierID)
	if supplierID == "" {
		supplierID = req.SupplierID
	}

	summary, err := processPurchaseReturnItemsTx(ctx, tx, CreatePurchaseReturnInput{
		BusinessID:      req.BusinessID,
		LocationID:      req.LocationID,
		SupplierID:      supplierID,
		ReferenceNumber: existing.ReferenceNumber,
		ReturnDate:      existing.ReturnDate,
		Status:          existing.Status,
		PaymentStatus:   existing.PaymentStatus,
		ReturnReason:    req.ReturnReason,
		Notes:           req.Notes,
		CreatedBy:       req.UpdatedBy,
		Items:           req.Items,
	}, req.PurchaseReturnID)
	if err != nil {
		return nil, err
	}

	paymentDue := summary.GrandTotal
	if _, err := tx.Exec(ctx, `
		UPDATE purchase_returns
		SET parent_purchase_id = NULLIF($2, '')::uuid,
		    parent_purchase_reference = $3,
		    grand_total = $4,
		    payment_due = $5,
		    total_quantity = $6,
		    items_count = $7
		WHERE id = $1::uuid
	`, req.PurchaseReturnID, summary.ParentPurchaseID, summary.ParentPurchaseReference, summary.GrandTotal, paymentDue, summary.TotalQuantity, summary.ItemsCount); err != nil {
		return nil, fmt.Errorf("update purchase return totals: %w", err)
	}

	updatedReturn, err := getPurchaseReturnByIDTx(ctx, tx, req.BusinessID, req.PurchaseReturnID)
	if err != nil {
		return nil, err
	}

	if err := insertPurchaseReturnLog(ctx, tx, CreatePurchaseReturnLogInput{
		BusinessID:       req.BusinessID,
		PurchaseReturnID: req.PurchaseReturnID,
		Action:           "updated",
		ActionedBy:       req.UpdatedBy,
		Note:             "Purchase return updated.",
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit purchase return update tx: %w", err)
	}

	return updatedReturn, nil
}

func DeletePurchaseReturnRepository(pool *pgxpool.Pool, businessID, purchaseReturnID, actionedBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	purchaseReturnID = strings.TrimSpace(purchaseReturnID)
	actionedBy = strings.TrimSpace(actionedBy)
	if businessID == "" || purchaseReturnID == "" {
		return ErrBusinessNotResolved
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin purchase return delete tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	returnEntry, err := getPurchaseReturnByIDTx(ctx, tx, businessID, purchaseReturnID)
	if err != nil {
		return err
	}

	items, err := listPurchaseReturnItemsTx(ctx, tx, businessID, purchaseReturnID)
	if err != nil {
		return err
	}

	for _, item := range items {
		balanceSnap, err := getOrCreateInventoryBalanceTx(ctx, tx, businessID, item.ProductID, item.LocationID, item.Quantity)
		if err != nil {
			return err
		}

		if item.InventoryBatchID != "" {
			if _, err := tx.Exec(ctx, `
				UPDATE inventory_batches
				SET quantity_remaining = quantity_remaining + $2,
				    updated_at = CURRENT_TIMESTAMP
				WHERE id = $1::uuid
			`, item.InventoryBatchID, item.Quantity); err != nil {
				return fmt.Errorf("restore purchase return inventory batch: %w", err)
			}
		}

		if err := insertStockMovementTx(ctx, tx, inventoryMovementInput{
			BusinessID:         businessID,
			PurchaseOrderID:    item.PurchaseOrderID,
			ReferenceNumber:    returnEntry.ReferenceNumber,
			LocationID:         item.LocationID,
			ProductID:          item.ProductID,
			InventoryBalanceID: balanceSnap.ID,
			InventoryBatchID:   item.InventoryBatchID,
			MovementType:       "purchase_receipt",
			QuantityIn:         item.Quantity,
			QuantityOut:        0,
			UnitCost:           item.UnitPrice,
			StockBefore:        balanceSnap.QuantityAvailable,
			StockAfter:         balanceSnap.QuantityAvailable + item.Quantity,
			Note:               fmt.Sprintf("Reversed purchase return for %s.", item.ProductName),
			PerformedBy:        actionedBy,
		}); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE purchase_return_items
		SET deleted = TRUE,
		    deleted_at = CURRENT_TIMESTAMP
		WHERE purchase_return_id = $1::uuid
		  AND deleted_at IS NULL
	`, purchaseReturnID); err != nil {
		return fmt.Errorf("soft delete purchase return items: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE purchase_returns
		SET deleted = TRUE,
		    deleted_at = CURRENT_TIMESTAMP
		WHERE business_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
	`, businessID, purchaseReturnID); err != nil {
		return fmt.Errorf("soft delete purchase return: %w", err)
	}

	if err := insertPurchaseReturnLog(ctx, tx, CreatePurchaseReturnLogInput{
		BusinessID:       businessID,
		PurchaseReturnID: purchaseReturnID,
		Action:           "deleted",
		ActionedBy:       actionedBy,
		Note:             "Purchase return deleted.",
	}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit purchase return delete tx: %w", err)
	}

	return nil
}

func getPurchaseReturnByIDTx(ctx context.Context, querier purchaseReturnTx, businessID, purchaseReturnID string) (*PurchaseReturn, error) {
	row := querier.QueryRow(ctx, purchaseReturnSelectQuery()+`
		WHERE pr.business_id = $1::uuid
		  AND pr.id = $2::uuid
		  AND pr.deleted_at IS NULL
		LIMIT 1
	`, businessID, purchaseReturnID)

	entry, err := scanPurchaseReturn(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPurchaseReturnNotFound
		}
		return nil, err
	}

	items, err := listPurchaseReturnItemsTx(ctx, querier, businessID, purchaseReturnID)
	if err != nil {
		return nil, err
	}
	entry.Items = items

	activities, err := listPurchaseReturnLogsByReturnIDTx(ctx, querier, businessID, purchaseReturnID)
	if err != nil {
		return nil, err
	}
	entry.Activities = activities

	return &entry, nil
}

func getPurchaseReturnByID(ctx context.Context, querier purchaseReturnTx, businessID, purchaseReturnID string) (*PurchaseReturn, error) {
	return getPurchaseReturnByIDTx(ctx, querier, businessID, purchaseReturnID)
}

type purchaseReturnAllocation struct {
	BatchID           string
	ProductName       string
	SKU               string
	SupplierID        string
	SupplierName      string
	LocationID        string
	LocationName      string
	PurchaseOrderID   string
	SourceReference   string
	LotNumber         string
	BatchNumber       string
	ExpiryDate        string
	ManufactureDate   string
	QuantityRemaining float64
}

func loadPurchaseReturnAllocationsTx(ctx context.Context, tx purchaseReturnTx, businessID, productID, locationID, supplierID string) ([]purchaseReturnAllocation, error) {
	args := []any{businessID, productID, locationID}
	supplierFilter := ""
	if strings.TrimSpace(supplierID) != "" {
		args = append(args, supplierID)
		supplierFilter = fmt.Sprintf("  AND COALESCE(ib.supplier_id::text, po.supplier_id::text, '') = $%d\n", len(args))
	}

	query := fmt.Sprintf(`
		SELECT
			ib.id::text,
			p.name,
			COALESCE(p.sku, ''),
			COALESCE(ib.supplier_id::text, po.supplier_id::text, ''),
			COALESCE(
				NULLIF(bs.business_name, ''),
				NULLIF(TRIM(CONCAT_WS(' ', bs.prefix, bs.first_name, bs.middle_name, bs.last_name)), ''),
				COALESCE(bs.contact_id, '')
			),
			ib.location_id::text,
			COALESCE(loc.location_name, ''),
			COALESCE(ib.source_id::text, ''),
			COALESCE(po.reference_number, ''),
			COALESCE(ib.lot_number, ''),
			COALESCE(ib.batch_number, ''),
			COALESCE(ib.expiry_date::text, ''),
			COALESCE(ib.manufacture_date::text, ''),
			COALESCE(ib.quantity_remaining, 0)
		FROM inventory_batches ib
		JOIN products p ON p.id = ib.product_id
		LEFT JOIN purchase_orders po ON po.id = ib.source_id::uuid AND ib.source_type = 'purchase_order'
		LEFT JOIN business_suppliers bs ON bs.id = COALESCE(ib.supplier_id, po.supplier_id) AND bs.business_id = ib.business_id AND bs.deleted_at IS NULL
		LEFT JOIN business_locations loc ON loc.id = ib.location_id
		WHERE ib.business_id = $1::uuid
		  AND ib.product_id = $2::uuid
		  AND ib.location_id = $3::uuid
		  AND ib.quantity_remaining > 0
%s
			ORDER BY COALESCE(ib.expiry_date, DATE '9999-12-31') ASC, ib.received_at ASC, ib.created_at ASC, ib.id ASC
			FOR UPDATE OF ib
	`, supplierFilter)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("load purchase return allocations: %w", err)
	}
	defer rows.Close()

	allocations := make([]purchaseReturnAllocation, 0)
	for rows.Next() {
		var allocation purchaseReturnAllocation
		if err := rows.Scan(
			&allocation.BatchID,
			&allocation.ProductName,
			&allocation.SKU,
			&allocation.SupplierID,
			&allocation.SupplierName,
			&allocation.LocationID,
			&allocation.LocationName,
			&allocation.PurchaseOrderID,
			&allocation.SourceReference,
			&allocation.LotNumber,
			&allocation.BatchNumber,
			&allocation.ExpiryDate,
			&allocation.ManufactureDate,
			&allocation.QuantityRemaining,
		); err != nil {
			return nil, fmt.Errorf("scan purchase return allocation: %w", err)
		}
		allocations = append(allocations, allocation)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purchase return allocations: %w", err)
	}

	return allocations, nil
}

func listPurchaseReturnItemsTx(ctx context.Context, tx purchaseReturnTx, businessID, purchaseReturnID string) ([]PurchaseReturnItem, error) {
	rows, err := tx.Query(ctx, `
		SELECT
			pri.id::text,
			pri.purchase_return_id::text,
			pri.business_id::text,
			pri.product_id::text,
			pri.product_name,
			pri.sku,
			COALESCE(pri.supplier_id::text, ''),
			pri.supplier_name,
			pri.location_id::text,
			pri.location_name,
			COALESCE(pri.purchase_order_id::text, ''),
			COALESCE(pri.inventory_batch_id::text, ''),
			COALESCE(pri.lot_number, ''),
			COALESCE(pri.batch_number, ''),
			COALESCE(pri.expiry_date::text, ''),
			COALESCE(pri.manufacture_date::text, ''),
			pri.quantity,
			pri.unit_price,
			pri.line_total,
			pri.created_at::text,
			pri.updated_at::text
		FROM purchase_return_items pri
		WHERE pri.business_id = $1::uuid
		  AND pri.purchase_return_id = $2::uuid
		  AND pri.deleted_at IS NULL
		ORDER BY pri.created_at ASC
	`, businessID, purchaseReturnID)
	if err != nil {
		return nil, fmt.Errorf("list purchase return items: %w", err)
	}
	defer rows.Close()

	items := make([]PurchaseReturnItem, 0)
	for rows.Next() {
		var item PurchaseReturnItem
		if err := rows.Scan(
			&item.ID,
			&item.PurchaseReturnID,
			&item.BusinessID,
			&item.ProductID,
			&item.ProductName,
			&item.SKU,
			&item.SupplierID,
			&item.SupplierName,
			&item.LocationID,
			&item.LocationName,
			&item.PurchaseOrderID,
			&item.InventoryBatchID,
			&item.LotNumber,
			&item.BatchNumber,
			&item.ExpiryDate,
			&item.ManufactureDate,
			&item.Quantity,
			&item.UnitPrice,
			&item.LineTotal,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan purchase return item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purchase return items: %w", err)
	}

	return items, nil
}

func listPurchaseReturnLogsByReturnIDTx(ctx context.Context, querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, businessID, purchaseReturnID string) ([]PurchaseReturnLog, error) {
	rows, err := querier.Query(ctx, `
		SELECT
			prl.id::text,
			prl.business_id::text,
			prl.purchase_return_id::text,
			prl.action,
			COALESCE(u.id::text, ''),
			COALESCE(u.full_name, 'System'),
			COALESCE(prl.note, ''),
			prl.action_date::text
		FROM purchase_returns_logs prl
		LEFT JOIN users u ON u.id = prl.actioned_by
		WHERE prl.business_id = $1::uuid
		  AND prl.purchase_return_id = $2::uuid
		ORDER BY prl.action_date ASC
	`, businessID, purchaseReturnID)
	if err != nil {
		return nil, fmt.Errorf("load purchase return logs: %w", err)
	}
	defer rows.Close()

	activities := make([]PurchaseReturnLog, 0)
	for rows.Next() {
		var logEntry PurchaseReturnLog
		var actionedByID sql.NullString
		var actionedByName sql.NullString
		if err := rows.Scan(
			&logEntry.ID,
			&logEntry.BusinessID,
			&logEntry.PurchaseReturnID,
			&logEntry.Action,
			&actionedByID,
			&actionedByName,
			&logEntry.Note,
			&logEntry.ActionDate,
		); err != nil {
			return nil, fmt.Errorf("scan purchase return log: %w", err)
		}
		if actionedByID.Valid || actionedByName.Valid {
			logEntry.ActionedBy = &PurchaseOrderCreatedBy{
				ID:   actionedByID.String,
				Name: actionedByName.String,
			}
		}
		activities = append(activities, logEntry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purchase return logs: %w", err)
	}

	return activities, nil
}

func purchaseReturnSelectQuery() string {
	return `
		SELECT
			pr.id::text,
			pr.business_id::text,
			COALESCE(pr.parent_purchase_id::text, ''),
			COALESCE(pr.parent_purchase_reference, ''),
			pr.reference_number,
			pr.return_date::text,
			pr.location_id::text,
			COALESCE(loc.location_name, ''),
			COALESCE(pr.supplier_id::text, ''),
			COALESCE(
				NULLIF(bs.business_name, ''),
				NULLIF(TRIM(CONCAT_WS(' ', bs.prefix, bs.first_name, bs.middle_name, bs.last_name)), ''),
				COALESCE(bs.contact_id, '')
			),
			pr.status,
			pr.payment_status,
			pr.grand_total,
			pr.payment_due,
			COALESCE(pr.return_reason, ''),
			COALESCE(pr.notes, ''),
			pr.items_count,
			pr.total_quantity,
			COALESCE(u.id::text, ''),
			COALESCE(u.full_name, 'System'),
			pr.created_at::text,
			pr.updated_at::text
		FROM purchase_returns pr
		LEFT JOIN business_locations loc ON loc.id = pr.location_id
		LEFT JOIN business_suppliers bs ON bs.id = pr.supplier_id AND bs.business_id = pr.business_id AND bs.deleted_at IS NULL
		LEFT JOIN users u ON u.id = pr.created_by`
}

func purchaseReturnListQuery(businessID string, filters ListPurchaseReturnsFilters) (string, []any) {
	query := purchaseReturnSelectQuery()
	conditions := []string{
		"pr.business_id = $1",
		"pr.deleted_at IS NULL",
	}
	args := []any{businessID}

	addCondition := func(condition string, values ...any) {
		updated := condition
		for _, value := range values {
			args = append(args, value)
			placeholder := fmt.Sprintf("$%d", len(args))
			updated = strings.Replace(updated, "?", placeholder, 1)
		}
		conditions = append(conditions, updated)
	}

	if value := strings.TrimSpace(filters.LocationID); value != "" {
		addCondition("pr.location_id::text = ?", value)
	}
	if value := strings.TrimSpace(filters.SupplierID); value != "" {
		addCondition("pr.supplier_id::text = ?", value)
	}
	if value := strings.TrimSpace(filters.Status); value != "" {
		addCondition("pr.status = ?", strings.ToLower(value))
	}
	if value := strings.TrimSpace(filters.PaymentStatus); value != "" {
		addCondition("pr.payment_status = ?", strings.ToLower(value))
	}
	if value := strings.TrimSpace(filters.DateFrom); value != "" {
		addCondition("pr.return_date::date >= ?::date", value)
	}
	if value := strings.TrimSpace(filters.DateTo); value != "" {
		addCondition("pr.return_date::date <= ?::date", value)
	}
	if value := strings.TrimSpace(filters.SearchQuery); value != "" {
		search := "%" + strings.ToLower(value) + "%"
		addCondition(`(
			LOWER(pr.reference_number) LIKE ?
			OR LOWER(COALESCE(pr.parent_purchase_reference, '')) LIKE ?
			OR LOWER(COALESCE(bs.business_name, '')) LIKE ?
			OR LOWER(COALESCE(TRIM(CONCAT_WS(' ', bs.prefix, bs.first_name, bs.middle_name, bs.last_name)), '')) LIKE ?
			OR LOWER(COALESCE(loc.location_name, '')) LIKE ?
		)`, search, search, search, search, search)
	}

	if len(conditions) > 0 {
		query += "\n\t\tWHERE " + strings.Join(conditions, "\n\t\t  AND ")
	}
	query += "\n\t\tORDER BY pr.return_date DESC, pr.created_at DESC"
	return query, args
}

func scanPurchaseReturn(scanner interface {
	Scan(dest ...any) error
}) (PurchaseReturn, error) {
	var (
		entry             PurchaseReturn
		parentPurchaseID  sql.NullString
		parentPurchaseRef sql.NullString
		supplierID        sql.NullString
		createdByID       sql.NullString
		createdByName     sql.NullString
		returnReason      sql.NullString
		notes             sql.NullString
	)

	if err := scanner.Scan(
		&entry.ID,
		&entry.BusinessID,
		&parentPurchaseID,
		&parentPurchaseRef,
		&entry.ReferenceNumber,
		&entry.ReturnDate,
		&entry.LocationID,
		&entry.LocationName,
		&supplierID,
		&entry.SupplierName,
		&entry.Status,
		&entry.PaymentStatus,
		&entry.GrandTotal,
		&entry.PaymentDue,
		&returnReason,
		&notes,
		&entry.ItemsCount,
		&entry.TotalQuantity,
		&createdByID,
		&createdByName,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	); err != nil {
		return PurchaseReturn{}, err
	}

	if parentPurchaseID.Valid {
		entry.ParentPurchaseID = parentPurchaseID.String
	}
	if parentPurchaseRef.Valid {
		entry.ParentPurchaseReference = parentPurchaseRef.String
	}
	if supplierID.Valid {
		entry.SupplierID = supplierID.String
	}
	if returnReason.Valid {
		entry.ReturnReason = returnReason.String
	}
	if notes.Valid {
		entry.Notes = notes.String
	}
	if createdByName.Valid && strings.TrimSpace(createdByName.String) != "" {
		entry.CreatedBy = &PurchaseOrderCreatedBy{
			ID:   createdByID.String,
			Name: createdByName.String,
		}
	}

	return entry, nil
}

func groupPurchaseReturnableStockItems(items []PurchaseReturnableStockItem) []PurchaseReturnableStockGroup {
	type groupedAccumulator struct {
		group PurchaseReturnableStockGroup
	}

	groups := make(map[string]*groupedAccumulator)
	order := make([]string, 0)

	for _, item := range items {
		groupKey := fmt.Sprintf("%s::%s", item.ProductID, item.SupplierID)
		accumulator, ok := groups[groupKey]
		if !ok {
			accumulator = &groupedAccumulator{
				group: PurchaseReturnableStockGroup{
					GroupKey:               groupKey,
					ProductID:              item.ProductID,
					ProductName:            item.ProductName,
					SKU:                    item.SKU,
					SupplierID:             item.SupplierID,
					SupplierName:           item.SupplierName,
					LocationName:           item.LocationName,
					SuppliedBySupplier:     0,
					SoldAlreadyForSupplier: 0,
					AvailableQuantity:      0,
					UnitPrice:              item.UnitPrice,
				},
			}
			groups[groupKey] = accumulator
			order = append(order, groupKey)
		}

		accumulator.group.SuppliedBySupplier += item.SuppliedBySupplier
		accumulator.group.SoldAlreadyForSupplier += item.SoldAlreadyForSupplier
		accumulator.group.AvailableQuantity += item.AvailableQuantity
		accumulator.group.UnitPrice = item.UnitPrice
	}

	grouped := make([]PurchaseReturnableStockGroup, 0, len(order))
	for _, key := range order {
		grouped = append(grouped, groups[key].group)
	}

	sort.SliceStable(grouped, func(i, j int) bool {
		if grouped[i].ProductName == grouped[j].ProductName {
			return grouped[i].SupplierName < grouped[j].SupplierName
		}
		return grouped[i].ProductName < grouped[j].ProductName
	})

	return grouped
}
