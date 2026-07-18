package sales

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type saleInventoryTx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type saleInventoryBalanceSnapshot struct {
	ID                string
	QuantityAvailable float64
	QuantityReserved  float64
}

type saleInventoryBatchSnapshot struct {
	ID                string
	QuantityRemaining float64
}

type saleInventoryBatchAllocation struct {
	BatchID    string
	Quantity   float64
	UnitCost   float64
	LineTotal  float64
	ExpiryDate *time.Time
	ReceivedAt time.Time
	CreatedAt  time.Time
}

func getOrCreateSaleInventoryBalanceTx(ctx context.Context, tx saleInventoryTx, businessID, productID, locationID string) (saleInventoryBalanceSnapshot, error) {
	var balance saleInventoryBalanceSnapshot
	err := tx.QueryRow(ctx, `
		SELECT id::text, COALESCE(quantity_available, 0), COALESCE(quantity_reserved, 0)
		FROM inventory_balances
		WHERE business_id = $1::uuid
		  AND product_id = $2::uuid
		  AND location_id = $3::uuid
		FOR UPDATE
	`, businessID, productID, locationID).Scan(&balance.ID, &balance.QuantityAvailable, &balance.QuantityReserved)
	if err == nil {
		return balance, nil
	}
	if err != pgx.ErrNoRows {
		return saleInventoryBalanceSnapshot{}, fmt.Errorf("load inventory balance: %w", err)
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
		return saleInventoryBalanceSnapshot{}, fmt.Errorf("insert inventory balance: %w", err)
	}
	return saleInventoryBalanceSnapshot{ID: balance.ID}, nil
}

func selectSaleInventoryBatchesTx(ctx context.Context, tx saleInventoryTx, businessID, productID, locationID, method string) ([]saleInventoryBatchSnapshot, error) {
	orderBy := "COALESCE(ib.expiry_date, DATE '9999-12-31') ASC, ib.received_at ASC, ib.created_at ASC, ib.id ASC"
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "LIFO":
		orderBy = "COALESCE(ib.expiry_date, DATE '0001-01-01') DESC, ib.received_at DESC, ib.created_at DESC, ib.id DESC"
	case "FEFO":
		orderBy = "COALESCE(ib.expiry_date, DATE '9999-12-31') ASC, ib.received_at ASC, ib.created_at ASC, ib.id ASC"
	case "FIFO":
		orderBy = "ib.received_at ASC, ib.created_at ASC, ib.id ASC"
	}

	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT ib.id::text, COALESCE(ib.quantity_remaining, 0)
		FROM inventory_batches ib
		WHERE business_id = $1::uuid
		  AND product_id = $2::uuid
		  AND location_id = $3::uuid
		  AND ib.quantity_remaining > 0
		ORDER BY %s
		FOR UPDATE
	`, orderBy), businessID, productID, locationID)
	if err != nil {
		return nil, fmt.Errorf("load sale inventory batches: %w", err)
	}
	defer rows.Close()

	batches := make([]saleInventoryBatchSnapshot, 0)
	for rows.Next() {
		var batch saleInventoryBatchSnapshot
		if err := rows.Scan(&batch.ID, &batch.QuantityRemaining); err != nil {
			return nil, fmt.Errorf("scan sale inventory batch: %w", err)
		}
		batches = append(batches, batch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sale inventory batches: %w", err)
	}

	return batches, nil
}

func insertSaleStockMovementTx(ctx context.Context, tx saleInventoryTx, input saleStockMovementInput) error {
	input.BusinessID = strings.TrimSpace(input.BusinessID)
	input.SaleID = strings.TrimSpace(input.SaleID)
	input.LocationID = strings.TrimSpace(input.LocationID)
	input.ProductID = strings.TrimSpace(input.ProductID)
	input.InventoryBalanceID = strings.TrimSpace(input.InventoryBalanceID)
	input.InventoryBatchID = strings.TrimSpace(input.InventoryBatchID)
	input.ReferenceNumber = strings.TrimSpace(input.ReferenceNumber)
	input.Note = strings.TrimSpace(input.Note)
	input.PerformedBy = strings.TrimSpace(input.PerformedBy)

	if input.BusinessID == "" || input.SaleID == "" || input.LocationID == "" || input.ProductID == "" {
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
			NULLIF($4, '')::uuid,
			NULLIF($5, '')::uuid,
			$6,
			'sale',
			$7::uuid,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			NULLIF($15, '')::uuid,
			CURRENT_TIMESTAMP,
			CURRENT_TIMESTAMP
		)
	`, input.BusinessID, input.ProductID, input.LocationID, input.InventoryBalanceID, input.InventoryBatchID, input.MovementType, input.SaleID, input.ReferenceNumber, input.QuantityIn, input.QuantityOut, input.UnitCost, input.StockBefore, input.StockAfter, input.Note, input.PerformedBy)
	if err != nil {
		return fmt.Errorf("insert sale stock movement: %w", err)
	}

	return nil
}

type saleStockMovementInput struct {
	BusinessID         string
	SaleID             string
	ReferenceNumber    string
	LocationID         string
	ProductID          string
	InventoryBalanceID string
	InventoryBatchID   string
	MovementType       string
	QuantityIn         float64
	QuantityOut        float64
	UnitCost           float64
	StockBefore        float64
	StockAfter         float64
	Note               string
	PerformedBy        string
}

func applySaleInventoryAllocationTx(
	ctx context.Context,
	tx saleInventoryTx,
	req CreateSaleOrderInput,
	item CreateSaleItemInput,
	saleID, saleItemID, productName, sku, unit string,
	allocationMode string,
) error {
	if item.Quantity <= 0 {
		return nil
	}

	if strings.TrimSpace(allocationMode) == "" {
		allocationMode = "none"
	}

	if allocationMode != "reserve" && allocationMode != "consume" {
		return nil
	}

	balance, err := getOrCreateSaleInventoryBalanceTx(ctx, tx, req.BusinessID, item.ProductID, req.LocationID)
	if err != nil {
		return err
	}

	batches, err := selectSaleInventoryBatchesTx(ctx, tx, req.BusinessID, item.ProductID, req.LocationID, req.StockAccountingMethod)
	if err != nil {
		return err
	}

	available := 0.0
	for _, batch := range batches {
		available += batch.QuantityRemaining
	}
	if available+balance.QuantityReserved < item.Quantity {
		return fmt.Errorf("insufficient stock for product %s", item.ProductID)
	}

	remaining := item.Quantity
	for idx, batch := range batches {
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

		nextRemaining := batch.QuantityRemaining - allocate
		if _, err := tx.Exec(ctx, `
			UPDATE inventory_batches
			SET quantity_remaining = $2,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $1::uuid
		`, batch.ID, nextRemaining); err != nil {
			return fmt.Errorf("update inventory batch: %w", err)
		}

		nextAvailable := balance.QuantityAvailable - allocate
		nextReserved := balance.QuantityReserved
		if allocationMode == "reserve" {
			nextReserved += allocate
		}
		if nextAvailable < 0 {
			nextAvailable = 0
		}
		if _, err := tx.Exec(ctx, `
			UPDATE inventory_balances
			SET quantity_available = $2,
			    quantity_reserved = $3,
			    last_movement_at = CURRENT_TIMESTAMP,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $1::uuid
		`, balance.ID, nextAvailable, nextReserved); err != nil {
			return fmt.Errorf("update inventory balance: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO sale_item_batch_allocations (
				sale_id,
				sale_item_id,
				business_id,
				inventory_batch_id,
				allocated_quantity,
				unit_cost,
				line_total,
				sort_order
			)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8)
		`, saleID, saleItemID, req.BusinessID, batch.ID, allocate, item.UnitCost, allocate*item.UnitPrice, idx); err != nil {
			return fmt.Errorf("insert sale batch allocation: %w", err)
		}

		if allocationMode == "consume" {
			if err := insertSaleStockMovementTx(ctx, tx, saleStockMovementInput{
				BusinessID:         req.BusinessID,
				SaleID:             saleID,
				ReferenceNumber:    req.ReferenceNumber,
				LocationID:         req.LocationID,
				ProductID:          item.ProductID,
				InventoryBalanceID: balance.ID,
				InventoryBatchID:   batch.ID,
				MovementType:       "sale",
				QuantityIn:         0,
				QuantityOut:        allocate,
				UnitCost:           item.UnitCost,
				StockBefore:        balance.QuantityAvailable + balance.QuantityReserved,
				StockAfter:         nextAvailable + nextReserved,
				Note:               "Recorded sale stock consumption.",
				PerformedBy:        req.CreatedBy,
			}); err != nil {
				return err
			}
		}

		balance.QuantityAvailable = nextAvailable
		balance.QuantityReserved = nextReserved
		remaining -= allocate
	}

	if remaining > 0 {
		return fmt.Errorf("insufficient stock for product %s", item.ProductID)
	}

	_ = productName
	_ = sku
	_ = unit
	return nil
}
