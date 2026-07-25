package stocktransfer

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type stockTransferTx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type transferBatchSnapshot struct {
	ID                string
	QuantityRemaining float64
	UnitCost          float64
	LotNumber         string
	BatchNumber       string
	ExpiryDate        sql.NullString
	ReceivedAt        string
	CreatedAt         string
}

type transferBalanceSnapshot struct {
	ID                string
	QuantityAvailable float64
}

func CreateStockTransferRepository(pool *pgxpool.Pool, req CreateStockTransferInput) (*StockTransfer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.ReferenceNumber = strings.TrimSpace(req.ReferenceNumber)
	req.TransferDate = strings.TrimSpace(req.TransferDate)
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	req.TransferType = strings.ToLower(strings.TrimSpace(req.TransferType))
	req.LocationFromID = strings.TrimSpace(req.LocationFromID)
	req.LocationToID = strings.TrimSpace(req.LocationToID)
	req.LocationFromName = strings.TrimSpace(req.LocationFromName)
	req.LocationToName = strings.TrimSpace(req.LocationToName)
	req.Currency = strings.TrimSpace(req.Currency)
	req.Notes = strings.TrimSpace(req.Notes)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)
	req.StockAccountingMethod = strings.TrimSpace(req.StockAccountingMethod)

	if req.BusinessID == "" || req.TransferDate == "" || req.LocationFromID == "" || req.LocationToID == "" || len(req.Items) == 0 {
		return nil, ErrInvalidStockTransferInput
	}
	if req.Status == "" {
		req.Status = "draft"
	}
	if req.TransferType == "" {
		req.TransferType = "local"
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.StockAccountingMethod == "" {
		req.StockAccountingMethod = "FIFO"
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin stock transfer tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if req.ReferenceNumber == "" {
		req.ReferenceNumber, err = generateStockTransferReferenceNumberTx(ctx, tx, req.BusinessID)
		if err != nil {
			return nil, err
		}
	}

	var stockTransferID string
	subtotal := 0.0
	totalQuantity := 0.0
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			return nil, ErrInvalidStockTransferInput
		}
		lineTotal := item.Quantity * item.UnitPrice
		subtotal += lineTotal
		totalQuantity += item.Quantity
	}
	totalAmount := subtotal + req.ShippingCharges

	if err := tx.QueryRow(ctx, `
		INSERT INTO stock_transfers (
			business_id,
			reference_number,
			transfer_date,
			status,
			transfer_type,
			source_location_id,
			destination_location_id,
			currency_code,
			shipping_charges,
			subtotal,
			total_amount,
			items_count,
			total_quantity,
			notes,
			created_by
		) VALUES (
			$1::uuid,
			$2,
			$3::timestamptz,
			$4,
			$5,
			$6::uuid,
			$7::uuid,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			NULLIF($15, '')::uuid
		)
		RETURNING id::text
	`, req.BusinessID, req.ReferenceNumber, req.TransferDate, req.Status, req.TransferType, req.LocationFromID, req.LocationToID, req.Currency, req.ShippingCharges, subtotal, totalAmount, len(req.Items), totalQuantity, req.Notes, req.CreatedBy).Scan(&stockTransferID); err != nil {
		return nil, fmt.Errorf("insert stock transfer: %w", err)
	}

	items := make([]StockTransferItem, 0, len(req.Items))
	for idx, item := range req.Items {
		productName, sku, unitName, err := loadTransferProductSnapshotTx(ctx, tx, req.BusinessID, item.ProductID)
		if err != nil {
			return nil, err
		}

		var stockTransferItemID string
		lineTotal := item.Quantity * item.UnitPrice
		if err := tx.QueryRow(ctx, `
			INSERT INTO stock_transfer_items (
				stock_transfer_id,
				business_id,
				product_id,
				product_name,
				sku,
				unit,
				quantity,
				unit_cost,
				line_total,
				sort_order
			) VALUES (
				$1::uuid,
				$2::uuid,
				$3::uuid,
				$4,
				$5,
				$6,
				$7,
				$8,
				$9,
				$10
			)
			RETURNING id::text
		`, stockTransferID, req.BusinessID, item.ProductID, productName, sku, unitName, item.Quantity, item.UnitPrice, lineTotal, idx).Scan(&stockTransferItemID); err != nil {
			return nil, fmt.Errorf("insert stock transfer item: %w", err)
		}

		transferItem := StockTransferItem{
			ID:              stockTransferItemID,
			StockTransferID: stockTransferID,
			BusinessID:      req.BusinessID,
			ProductID:       item.ProductID,
			ProductName:     productName,
			SKU:             sku,
			Unit:            unitName,
			Quantity:        item.Quantity,
			UnitPrice:       item.UnitPrice,
			Subtotal:        lineTotal,
			SortOrder:       idx,
		}

		if err := applyTransferInventoryTx(ctx, tx, req, stockTransferID, transferItem, item); err != nil {
			return nil, err
		}

		items = append(items, transferItem)
	}

	created, err := GetStockTransferByIDRepositoryTx(ctx, tx, req.BusinessID, stockTransferID)
	if err != nil {
		return nil, err
	}

	if len(created.Items) == 0 {
		created.Items = items
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit stock transfer tx: %w", err)
	}

	return created, nil
}

func ListStockTransfersRepository(pool *pgxpool.Pool, businessID string) ([]StockTransferListItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	rows, err := pool.Query(ctx, `
		SELECT
			st.id::text,
			st.reference_number,
			st.transfer_date::text,
			st.status,
			st.source_location_id::text,
			st.destination_location_id::text,
			COALESCE(sl.location_name, ''),
			COALESCE(dl.location_name, ''),
			COALESCE(st.currency_code, 'USD'),
			COALESCE(st.shipping_charges, 0),
			COALESCE(st.subtotal, 0),
			COALESCE(st.total_amount, 0),
			COALESCE(st.items_count, 0),
			COALESCE(st.total_quantity, 0),
			COALESCE(st.notes, ''),
			COALESCE(st.transfer_type, 'local'),
			st.created_at::text,
			st.updated_at::text
		FROM stock_transfers st
		LEFT JOIN business_locations sl ON sl.id = st.source_location_id
		LEFT JOIN business_locations dl ON dl.id = st.destination_location_id
		WHERE st.business_id = $1::uuid
		  AND st.deleted_at IS NULL
		ORDER BY st.transfer_date DESC, st.created_at DESC
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list stock transfers: %w", err)
	}
	defer rows.Close()

	items := make([]StockTransferListItem, 0)
	for rows.Next() {
		var item StockTransferListItem
		if err := rows.Scan(
			&item.ID,
			&item.ReferenceNumber,
			&item.TransferDate,
			&item.Status,
			&item.LocationFromID,
			&item.LocationToID,
			&item.LocationFromName,
			&item.LocationToName,
			&item.Currency,
			&item.ShippingCharges,
			&item.Subtotal,
			&item.TotalAmount,
			&item.ItemsCount,
			&item.TotalQuantity,
			&item.Notes,
			&item.Type,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stock transfer: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stock transfers: %w", err)
	}

	return items, nil
}

func GetStockTransferByIDRepository(pool *pgxpool.Pool, businessID, stockTransferID string) (*StockTransfer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return GetStockTransferByIDRepositoryTx(ctx, pool, businessID, stockTransferID)
}

func GetStockTransferByIDRepositoryTx(ctx context.Context, tx stockTransferTx, businessID, stockTransferID string) (*StockTransfer, error) {
	businessID = strings.TrimSpace(businessID)
	stockTransferID = strings.TrimSpace(stockTransferID)
	if businessID == "" || stockTransferID == "" {
		return nil, ErrBusinessNotResolved
	}

	var transfer StockTransfer
	if err := tx.QueryRow(ctx, `
		SELECT
			id::text,
			business_id::text,
			reference_number,
			transfer_date::text,
			status,
			source_location_id::text,
			destination_location_id::text,
			COALESCE(sl.location_name, ''),
			COALESCE(dl.location_name, ''),
			COALESCE(currency_code, 'USD'),
			COALESCE(shipping_charges, 0),
			COALESCE(subtotal, 0),
			COALESCE(total_amount, 0),
			COALESCE(items_count, 0),
			COALESCE(total_quantity, 0),
			COALESCE(notes, ''),
			COALESCE(transfer_type, 'local'),
			COALESCE(created_by::text, ''),
			COALESCE(approved_by::text, ''),
			COALESCE(approved_at::text, ''),
			COALESCE(completed_at::text, ''),
			created_at::text,
			updated_at::text
		FROM stock_transfers st
		LEFT JOIN business_locations sl ON sl.id = st.source_location_id
		LEFT JOIN business_locations dl ON dl.id = st.destination_location_id
		WHERE st.business_id = $1::uuid
		  AND st.id = $2::uuid
		  AND st.deleted_at IS NULL
		LIMIT 1
	`, businessID, stockTransferID).Scan(
		&transfer.ID,
		&transfer.BusinessID,
		&transfer.ReferenceNumber,
		&transfer.TransferDate,
		&transfer.Status,
		&transfer.LocationFromID,
		&transfer.LocationToID,
		&transfer.LocationFromName,
		&transfer.LocationToName,
		&transfer.Currency,
		&transfer.ShippingCharges,
		&transfer.Subtotal,
		&transfer.TotalAmount,
		&transfer.ItemsCount,
		&transfer.TotalQuantity,
		&transfer.Notes,
		&transfer.Type,
		&transfer.CreatedBy,
		&transfer.ApprovedBy,
		&transfer.ApprovedAt,
		&transfer.CompletedAt,
		&transfer.CreatedAt,
		&transfer.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrStockTransferNotFound
		}
		return nil, fmt.Errorf("load stock transfer: %w", err)
	}

	items, err := listStockTransferItemsTx(ctx, tx, stockTransferID)
	if err != nil {
		return nil, err
	}
	transfer.Items = items

	return &transfer, nil
}

func listStockTransferItemsTx(ctx context.Context, tx stockTransferTx, stockTransferID string) ([]StockTransferItem, error) {
	rows, err := tx.Query(ctx, `
		SELECT
			id::text,
			stock_transfer_id::text,
			business_id::text,
			product_id::text,
			COALESCE(product_name, ''),
			COALESCE(sku, ''),
			COALESCE(unit, ''),
			COALESCE(quantity, 0),
			COALESCE(unit_cost, 0),
			COALESCE(line_total, 0),
			sort_order,
			created_at::text,
			updated_at::text
		FROM stock_transfer_items
		WHERE stock_transfer_id = $1::uuid
		ORDER BY sort_order ASC, created_at ASC
	`, stockTransferID)
	if err != nil {
		return nil, fmt.Errorf("list stock transfer items: %w", err)
	}
	defer rows.Close()

	items := make([]StockTransferItem, 0)
	for rows.Next() {
		var item StockTransferItem
		if err := rows.Scan(
			&item.ID,
			&item.StockTransferID,
			&item.BusinessID,
			&item.ProductID,
			&item.ProductName,
			&item.SKU,
			&item.Unit,
			&item.Quantity,
			&item.UnitPrice,
			&item.Subtotal,
			&item.SortOrder,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stock transfer item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stock transfer items: %w", err)
	}

	return items, nil
}

func generateStockTransferReferenceNumberTx(ctx context.Context, tx stockTransferTx, businessID string) (string, error) {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, businessID); err != nil {
		return "", fmt.Errorf("lock stock transfer sequence: %w", err)
	}

	var count int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM stock_transfers
		WHERE business_id = $1::uuid
		  AND deleted_at IS NULL
	`, businessID).Scan(&count); err != nil {
		return "", fmt.Errorf("count stock transfers: %w", err)
	}

	return fmt.Sprintf("ST-%05d", count+1), nil
}

func loadTransferProductSnapshotTx(ctx context.Context, tx stockTransferTx, businessID, productID string) (string, string, string, error) {
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
			return "", "", "", fmt.Errorf("transfer product not found")
		}
		return "", "", "", fmt.Errorf("load transfer product: %w", err)
	}

	return productName.String, sku.String, unitName.String, nil
}

func applyTransferInventoryTx(ctx context.Context, tx stockTransferTx, req CreateStockTransferInput, transferID string, transferItem StockTransferItem, input CreateStockTransferItemInput) error {
	if !transferAffectsInventory(req.Status) {
		return nil
	}

	batches, err := selectTransferInventoryBatchesTx(ctx, tx, req.BusinessID, input.ProductID, req.LocationFromID, req.StockAccountingMethod)
	if err != nil {
		return err
	}

	available := 0.0
	for _, batch := range batches {
		available += batch.QuantityRemaining
	}
	if available < input.Quantity {
		return fmt.Errorf("insufficient stock for product %s", input.ProductID)
	}

	remaining := input.Quantity
	sortOrder := 0
	for _, batch := range batches {
		if remaining <= 0 {
			break
		}

		allocate := batch.QuantityRemaining
		if allocate > remaining {
			allocate = remaining
		}
		if allocate <= 0 {
			continue
		}

		if err := adjustTransferInventoryBalanceTx(ctx, tx, req.BusinessID, input.ProductID, req.LocationFromID, -allocate); err != nil {
			return err
		}
		if err := adjustTransferInventoryBalanceTx(ctx, tx, req.BusinessID, input.ProductID, req.LocationToID, allocate); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `
			UPDATE inventory_batches
			SET quantity_remaining = GREATEST(quantity_remaining - $1, 0),
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $2::uuid
		`, allocate, batch.ID); err != nil {
			return fmt.Errorf("decrement source inventory batch: %w", err)
		}

		var newBatchID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO inventory_batches (
				business_id,
				product_id,
				location_id,
				source_type,
				source_id,
				lot_number,
				batch_number,
				expiry_date,
				unit_cost,
				quantity_received,
				quantity_remaining,
				received_at,
				created_by,
				created_at,
				updated_at
			)
			VALUES (
				$1::uuid,
				$2::uuid,
				$3::uuid,
				'stock_transfer',
				$4::uuid,
				$5,
				$6,
				NULLIF($7, '')::date,
				$8,
				$9,
				$10,
				CURRENT_TIMESTAMP,
				NULLIF($11, '')::uuid,
				CURRENT_TIMESTAMP,
				CURRENT_TIMESTAMP
			)
			RETURNING id::text
		`, req.BusinessID, input.ProductID, req.LocationToID, transferID, batch.LotNumber, batch.BatchNumber, batch.ExpiryDate.String, batch.UnitCost, allocate, allocate, req.CreatedBy).Scan(&newBatchID); err != nil {
			return fmt.Errorf("insert destination inventory batch: %w", err)
		}

		if err := insertTransferStockMovementTx(ctx, tx, transferStockMovementInput{
			BusinessID:       req.BusinessID,
			TransferID:       transferID,
			ReferenceNumber:  req.ReferenceNumber,
			LocationID:       req.LocationFromID,
			ProductID:        input.ProductID,
			InventoryBatchID: batch.ID,
			MovementType:     "stock_transfer_out",
			QuantityIn:       0,
			QuantityOut:      allocate,
			UnitCost:         batch.UnitCost,
			StockBefore:      batch.QuantityRemaining,
			StockAfter:       batch.QuantityRemaining - allocate,
			Note:             fmt.Sprintf("Transferred %s to %s.", transferItem.ProductName, req.LocationToName),
			PerformedBy:      req.CreatedBy,
		}); err != nil {
			return err
		}

		if err := insertTransferStockMovementTx(ctx, tx, transferStockMovementInput{
			BusinessID:       req.BusinessID,
			TransferID:       transferID,
			ReferenceNumber:  req.ReferenceNumber,
			LocationID:       req.LocationToID,
			ProductID:        input.ProductID,
			InventoryBatchID: newBatchID,
			MovementType:     "stock_transfer_in",
			QuantityIn:       allocate,
			QuantityOut:      0,
			UnitCost:         batch.UnitCost,
			StockBefore:      0,
			StockAfter:       allocate,
			Note:             fmt.Sprintf("Received %s from %s.", transferItem.ProductName, req.LocationFromName),
			PerformedBy:      req.CreatedBy,
		}); err != nil {
			return err
		}

		if err := insertTransferBatchAllocationTx(ctx, tx, transferBatchAllocationInput{
			StockTransferID:     transferID,
			StockTransferItemID: transferItem.ID,
			BusinessID:          req.BusinessID,
			InventoryBatchID:    batch.ID,
			AllocatedQuantity:   allocate,
			UnitCost:            batch.UnitCost,
			LineTotal:           allocate * batch.UnitCost,
			SortOrder:           sortOrder,
		}); err != nil {
			return err
		}

		sortOrder++
		remaining -= allocate
	}

	return nil
}

type transferBatchAllocationInput struct {
	StockTransferID     string
	StockTransferItemID string
	BusinessID          string
	InventoryBatchID    string
	AllocatedQuantity   float64
	UnitCost            float64
	LineTotal           float64
	SortOrder           int
}

func insertTransferBatchAllocationTx(ctx context.Context, tx stockTransferTx, req transferBatchAllocationInput) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO stock_transfer_item_batch_allocations (
			stock_transfer_id,
			stock_transfer_item_id,
			business_id,
			inventory_batch_id,
			allocated_quantity,
			unit_cost,
			line_total,
			sort_order,
			created_at,
			updated_at
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			$3::uuid,
			$4::uuid,
			$5,
			$6,
			$7,
			$8,
			CURRENT_TIMESTAMP,
			CURRENT_TIMESTAMP
		)
	`, req.StockTransferID, req.StockTransferItemID, req.BusinessID, req.InventoryBatchID, req.AllocatedQuantity, req.UnitCost, req.LineTotal, req.SortOrder)
	if err != nil {
		return fmt.Errorf("insert stock transfer batch allocation: %w", err)
	}

	return nil
}

type transferStockMovementInput struct {
	BusinessID       string
	TransferID       string
	ReferenceNumber  string
	LocationID       string
	ProductID        string
	InventoryBatchID string
	MovementType     string
	QuantityIn       float64
	QuantityOut      float64
	UnitCost         float64
	StockBefore      float64
	StockAfter       float64
	Note             string
	PerformedBy      string
}

func insertTransferStockMovementTx(ctx context.Context, tx stockTransferTx, req transferStockMovementInput) error {
	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.TransferID = strings.TrimSpace(req.TransferID)
	req.ReferenceNumber = strings.TrimSpace(req.ReferenceNumber)
	req.LocationID = strings.TrimSpace(req.LocationID)
	req.ProductID = strings.TrimSpace(req.ProductID)
	req.InventoryBatchID = strings.TrimSpace(req.InventoryBatchID)
	req.MovementType = strings.TrimSpace(req.MovementType)
	req.Note = strings.TrimSpace(req.Note)
	req.PerformedBy = strings.TrimSpace(req.PerformedBy)

	if req.BusinessID == "" || req.TransferID == "" || req.LocationID == "" || req.ProductID == "" || req.MovementType == "" {
		return nil
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO stock_movements (
			business_id,
			product_id,
			location_id,
			inventory_balance_id,
			inventory_batch_id,
			movement_type,
			source_type,
			source_id,
			reference_number,
			quantity_in,
			quantity_out,
			unit_cost,
			stock_before,
			stock_after,
			note,
			performed_by,
			occurred_at,
			created_at
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			$3::uuid,
			NULL,
			NULLIF($4, '')::uuid,
			$5,
			'stock_transfer',
			$6::uuid,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13,
			NULLIF($14, '')::uuid,
			CURRENT_TIMESTAMP,
			CURRENT_TIMESTAMP
		)
	`, req.BusinessID, req.ProductID, req.LocationID, req.InventoryBatchID, req.MovementType, req.TransferID, req.ReferenceNumber, req.QuantityIn, req.QuantityOut, req.UnitCost, req.StockBefore, req.StockAfter, req.Note, req.PerformedBy)
	if err != nil {
		return fmt.Errorf("insert stock transfer stock movement: %w", err)
	}

	return nil
}

func transferAffectsInventory(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "approved", "processing", "completed":
		return true
	default:
		return false
	}
}

func selectTransferInventoryBatchesTx(ctx context.Context, tx stockTransferTx, businessID, productID, locationID, method string) ([]transferBatchSnapshot, error) {
	orderBy := "ib.received_at ASC, ib.created_at ASC, ib.id ASC"
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "LIFO":
		orderBy = "ib.received_at DESC, ib.created_at DESC, ib.id DESC"
	case "FEFO":
		orderBy = "COALESCE(ib.expiry_date, DATE '9999-12-31') ASC, ib.received_at ASC, ib.created_at ASC, ib.id ASC"
	}

	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT
			ib.id::text,
			COALESCE(ib.quantity_remaining, 0),
			COALESCE(ib.unit_cost, 0),
			COALESCE(ib.lot_number, ''),
			COALESCE(ib.batch_number, ''),
			COALESCE(ib.expiry_date::text, ''),
			COALESCE(ib.received_at::text, ''),
			COALESCE(ib.created_at::text, '')
		FROM inventory_batches ib
		WHERE ib.business_id = $1::uuid
		  AND ib.product_id = $2::uuid
		  AND ib.location_id = $3::uuid
		  AND COALESCE(ib.quantity_remaining, 0) > 0
		ORDER BY %s
		FOR UPDATE
	`, orderBy), businessID, productID, locationID)
	if err != nil {
		return nil, fmt.Errorf("list transfer inventory batches: %w", err)
	}
	defer rows.Close()

	batches := make([]transferBatchSnapshot, 0)
	for rows.Next() {
		var batch transferBatchSnapshot
		if err := rows.Scan(&batch.ID, &batch.QuantityRemaining, &batch.UnitCost, &batch.LotNumber, &batch.BatchNumber, &batch.ExpiryDate, &batch.ReceivedAt, &batch.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan transfer inventory batch: %w", err)
		}
		batches = append(batches, batch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transfer inventory batches: %w", err)
	}

	return batches, nil
}

func adjustTransferInventoryBalanceTx(ctx context.Context, tx stockTransferTx, businessID, productID, locationID string, delta float64) error {
	if delta == 0 {
		return nil
	}

	var balance transferBalanceSnapshot
	err := tx.QueryRow(ctx, `
		SELECT id::text, COALESCE(quantity_available, 0)
		FROM inventory_balances
		WHERE business_id = $1::uuid
		  AND product_id = $2::uuid
		  AND location_id = $3::uuid
		FOR UPDATE
	`, businessID, productID, locationID).Scan(&balance.ID, &balance.QuantityAvailable)
	if err == pgx.ErrNoRows {
		if delta < 0 {
			return fmt.Errorf("insufficient stock for product %s", productID)
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO inventory_balances (
				business_id,
				product_id,
				location_id,
				quantity_available,
				quantity_reserved,
				last_movement_at,
				created_at,
				updated_at
			)
			VALUES ($1::uuid, $2::uuid, $3::uuid, 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			RETURNING id::text
		`, businessID, productID, locationID).Scan(&balance.ID); err != nil {
			return fmt.Errorf("insert inventory balance: %w", err)
		}
		balance.QuantityAvailable = 0
	} else if err != nil {
		return fmt.Errorf("load inventory balance: %w", err)
	}

	nextQuantity := balance.QuantityAvailable + delta
	if nextQuantity < 0 {
		return fmt.Errorf("insufficient stock for product %s", productID)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE inventory_balances
		SET quantity_available = $4,
		    last_movement_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE business_id = $1::uuid
		  AND product_id = $2::uuid
		  AND location_id = $3::uuid
	`, businessID, productID, locationID, nextQuantity); err != nil {
		return fmt.Errorf("update inventory balance: %w", err)
	}

	return nil
}
