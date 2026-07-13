package purchaseorder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type purchaseOrderInventoryTx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type inventoryBalanceSnapshot struct {
	ID                string
	QuantityAvailable float64
}

type inventoryBatchSnapshot struct {
	ID                string
	QuantityRemaining  float64
}

func statusAffectsInventory(note string) bool {
	note = strings.ToLower(strings.TrimSpace(note))
	return strings.HasPrefix(note, "yes")
}

func purchaseOrderStatusAffectsInventory(ctx context.Context, querier purchaseOrderInventoryTx, statusCode string) (bool, error) {
	status, err := getPurchaseOrderStatusByCode(ctx, querier, statusCode)
	if err != nil {
		return false, nil
	}
	return statusAffectsInventory(status.StockAffectedNote), nil
}

func syncPurchaseOrderInventoryTx(
	ctx context.Context,
	tx purchaseOrderInventoryTx,
	req UpdatePurchaseOrderInput,
	existing *PurchaseOrder,
	existingItemReceipts map[string]float64,
	items []CreatePurchaseOrderItemInput,
) error {
	prevAffects, err := purchaseOrderStatusAffectsInventory(ctx, tx, existing.Status)
	if err != nil {
		return err
	}
	nextAffects, err := purchaseOrderStatusAffectsInventory(ctx, tx, req.Status)
	if err != nil {
		return err
	}

	if !prevAffects && !nextAffects {
		return nil
	}

	prevLocationID := strings.TrimSpace(existing.LocationID)
	nextLocationID := strings.TrimSpace(req.LocationID)
	if nextLocationID == "" {
		nextLocationID = prevLocationID
	}

	for _, item := range items {
		productID := strings.TrimSpace(item.ProductID)
		if productID == "" {
			continue
		}

		previousReceived := existingItemReceipts[productID]
		if previousReceived < 0 {
			previousReceived = 0
		}

		nextReceived := 0.0
		if item.ReceivedQuantity != nil {
			nextReceived = *item.ReceivedQuantity
		}
		if nextReceived < 0 {
			nextReceived = 0
		}
		if nextReceived > item.OrderQuantity {
			nextReceived = item.OrderQuantity
		}

		reverseQty := 0.0
		applyQty := 0.0

		switch {
		case prevAffects && nextAffects:
			if prevLocationID != nextLocationID {
				reverseQty = previousReceived
				applyQty = nextReceived
			} else {
				delta := nextReceived - previousReceived
				if delta > 0 {
					applyQty = delta
				} else {
					reverseQty = -delta
				}
			}
		case !prevAffects && nextAffects:
			applyQty = nextReceived
		case prevAffects && !nextAffects:
			reverseQty = previousReceived
		}

		if reverseQty > 0 {
			if err := applyInventoryDeltaTx(ctx, tx, inventoryDeltaInput{
				BusinessID:      req.BusinessID,
				PurchaseOrderID: req.PurchaseOrderID,
				ReferenceNumber: req.ReferenceNumber,
				LocationID:      prevLocationID,
				ProductID:       productID,
				QuantityDelta:   -reverseQty,
				UnitCost:        item.UnitCostBeforeTax,
				LotNumber:       item.LotNumber,
				ExpiryDate:      item.ExpiryDate,
				PerformedBy:     req.UpdatedBy,
				Note:            fmt.Sprintf("Reversed purchase order stock for %s.", item.ProductID),
			}); err != nil {
				return err
			}
		}

		if applyQty > 0 {
			if err := applyInventoryDeltaTx(ctx, tx, inventoryDeltaInput{
				BusinessID:      req.BusinessID,
				PurchaseOrderID: req.PurchaseOrderID,
				ReferenceNumber: req.ReferenceNumber,
				LocationID:      nextLocationID,
				ProductID:       productID,
				QuantityDelta:   applyQty,
				UnitCost:        item.UnitCostBeforeTax,
				LotNumber:       item.LotNumber,
				ExpiryDate:      item.ExpiryDate,
				PerformedBy:     req.UpdatedBy,
				Note:            fmt.Sprintf("Recorded purchase order receipt for %s.", item.ProductID),
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

type inventoryDeltaInput struct {
	BusinessID      string
	PurchaseOrderID  string
	ReferenceNumber string
	LocationID      string
	ProductID       string
	QuantityDelta   float64
	UnitCost        float64
	LotNumber       string
	ExpiryDate      string
	PerformedBy     string
	Note            string
}

func applyInventoryDeltaTx(ctx context.Context, tx purchaseOrderInventoryTx, req inventoryDeltaInput) error {
	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.PurchaseOrderID = strings.TrimSpace(req.PurchaseOrderID)
	req.ReferenceNumber = strings.TrimSpace(req.ReferenceNumber)
	req.LocationID = strings.TrimSpace(req.LocationID)
	req.ProductID = strings.TrimSpace(req.ProductID)
	req.LotNumber = strings.TrimSpace(req.LotNumber)
	req.ExpiryDate = strings.TrimSpace(req.ExpiryDate)
	req.PerformedBy = strings.TrimSpace(req.PerformedBy)
	req.Note = strings.TrimSpace(req.Note)

	if req.BusinessID == "" || req.PurchaseOrderID == "" || req.LocationID == "" || req.ProductID == "" || req.QuantityDelta == 0 {
		return nil
	}

	balance, err := getOrCreateInventoryBalanceTx(ctx, tx, req.BusinessID, req.ProductID, req.LocationID, req.QuantityDelta)
	if err != nil {
		return err
	}

	if req.QuantityDelta > 0 {
		batchID, err := insertInventoryBatchTx(ctx, tx, req, balance.ID)
		if err != nil {
			return err
		}

		return insertStockMovementTx(ctx, tx, inventoryMovementInput{
			BusinessID:          req.BusinessID,
			PurchaseOrderID:     req.PurchaseOrderID,
			ReferenceNumber:     req.ReferenceNumber,
			LocationID:          req.LocationID,
			ProductID:           req.ProductID,
			InventoryBalanceID:  balance.ID,
			InventoryBatchID:    batchID,
			MovementType:        "purchase_receipt",
			QuantityIn:          req.QuantityDelta,
			QuantityOut:         0,
			UnitCost:            req.UnitCost,
			StockBefore:         balance.QuantityAvailable,
			StockAfter:          balance.QuantityAvailable + req.QuantityDelta,
			Note:                req.Note,
			PerformedBy:         req.PerformedBy,
		})
	}

	consumed, err := consumeInventoryBatchesTx(ctx, tx, req.BusinessID, req.ProductID, req.LocationID, req.PurchaseOrderID, -req.QuantityDelta)
	if err != nil {
		return err
	}

	return insertStockMovementTx(ctx, tx, inventoryMovementInput{
		BusinessID:         req.BusinessID,
		PurchaseOrderID:    req.PurchaseOrderID,
		ReferenceNumber:    req.ReferenceNumber,
		LocationID:         req.LocationID,
		ProductID:          req.ProductID,
		InventoryBalanceID: balance.ID,
		InventoryBatchID:   consumed.batchID,
		MovementType:       "purchase_return",
		QuantityIn:         0,
		QuantityOut:        -req.QuantityDelta,
		UnitCost:           req.UnitCost,
		StockBefore:        balance.QuantityAvailable,
		StockAfter:         balance.QuantityAvailable + req.QuantityDelta,
		Note:               req.Note,
		PerformedBy:        req.PerformedBy,
	})
}

func getOrCreateInventoryBalanceTx(ctx context.Context, tx purchaseOrderInventoryTx, businessID, productID, locationID string, delta float64) (inventoryBalanceSnapshot, error) {
	var balance inventoryBalanceSnapshot
	err := tx.QueryRow(ctx, `
		SELECT id::text, COALESCE(quantity_available, 0)
		FROM inventory_balances
		WHERE business_id = $1::uuid
		  AND product_id = $2::uuid
		  AND location_id = $3::uuid
		FOR UPDATE
	`, businessID, productID, locationID).Scan(&balance.ID, &balance.QuantityAvailable)
	if err == nil {
		nextQuantity := balance.QuantityAvailable + delta
		if nextQuantity < 0 {
			return inventoryBalanceSnapshot{}, fmt.Errorf("inventory balance cannot go below zero for product %s", productID)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE inventory_balances
			SET quantity_available = $4,
			    last_movement_at = CURRENT_TIMESTAMP,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $1::uuid
			  AND business_id = $2::uuid
			  AND product_id = $3::uuid
		`, balance.ID, businessID, productID, nextQuantity); err != nil {
			return inventoryBalanceSnapshot{}, fmt.Errorf("update inventory balance: %w", err)
		}
		return inventoryBalanceSnapshot{ID: balance.ID, QuantityAvailable: balance.QuantityAvailable}, nil
	}

	if err != pgx.ErrNoRows {
		return inventoryBalanceSnapshot{}, fmt.Errorf("load inventory balance: %w", err)
	}

	if delta < 0 {
		return inventoryBalanceSnapshot{}, fmt.Errorf("inventory balance cannot go below zero for product %s", productID)
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
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id::text
	`, businessID, productID, locationID, delta).Scan(&balance.ID); err != nil {
		return inventoryBalanceSnapshot{}, fmt.Errorf("insert inventory balance: %w", err)
	}
	return inventoryBalanceSnapshot{ID: balance.ID, QuantityAvailable: 0}, nil
}

func insertInventoryBatchTx(ctx context.Context, tx purchaseOrderInventoryTx, req inventoryDeltaInput, inventoryBalanceID string) (string, error) {
	var expiry *time.Time
	if req.ExpiryDate != "" {
		parsedExpiry, err := time.Parse("2006-01-02", req.ExpiryDate)
		if err != nil {
			return "", fmt.Errorf("parse expiry date for inventory batch: %w", err)
		}
		expiry = &parsedExpiry
	}

	var batchID string
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
			'purchase_order',
			$4::uuid,
			$5,
			$6,
			$7,
			$8,
			$9,
			$9,
			CURRENT_TIMESTAMP,
			NULLIF($10, '')::uuid,
			CURRENT_TIMESTAMP,
			CURRENT_TIMESTAMP
		)
		RETURNING id::text
	`, req.BusinessID, req.ProductID, req.LocationID, req.PurchaseOrderID, req.LotNumber, req.ReferenceNumber, expiry, req.UnitCost, req.QuantityDelta, req.PerformedBy).Scan(&batchID); err != nil {
		return "", fmt.Errorf("insert inventory batch: %w", err)
	}
	_ = inventoryBalanceID
	return batchID, nil
}

type consumedBatchResult struct {
	batchID string
}

func consumeInventoryBatchesTx(ctx context.Context, tx purchaseOrderInventoryTx, businessID, productID, locationID, purchaseOrderID string, quantity float64) (consumedBatchResult, error) {
	if quantity <= 0 {
		return consumedBatchResult{}, nil
	}

	rows, err := tx.Query(ctx, `
		SELECT id::text, quantity_remaining
		FROM inventory_batches
		WHERE business_id = $1::uuid
		  AND product_id = $2::uuid
		  AND location_id = $3::uuid
		  AND source_type = 'purchase_order'
		  AND source_id = $4::uuid
		  AND quantity_remaining > 0
		ORDER BY received_at DESC, created_at DESC, id DESC
		FOR UPDATE
	`, businessID, productID, locationID, purchaseOrderID)
	if err != nil {
		return consumedBatchResult{}, fmt.Errorf("load inventory batches: %w", err)
	}
	defer rows.Close()

	batches := make([]inventoryBatchSnapshot, 0)
	for rows.Next() {
		var batch inventoryBatchSnapshot
		if err := rows.Scan(&batch.ID, &batch.QuantityRemaining); err != nil {
			return consumedBatchResult{}, fmt.Errorf("scan inventory batch: %w", err)
		}
		batches = append(batches, batch)
	}
	if err := rows.Err(); err != nil {
		return consumedBatchResult{}, fmt.Errorf("iterate inventory batches: %w", err)
	}

	remaining := quantity
	var lastBatchID string
	for _, batch := range batches {
		if remaining <= 0 {
			break
		}
		consume := batch.QuantityRemaining
		if consume > remaining {
			consume = remaining
		}
		nextRemaining := batch.QuantityRemaining - consume
		if nextRemaining < 0 {
			nextRemaining = 0
		}
		if _, err := tx.Exec(ctx, `
			UPDATE inventory_batches
			SET quantity_remaining = $2,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $1::uuid
		`, batch.ID, nextRemaining); err != nil {
			return consumedBatchResult{}, fmt.Errorf("update inventory batch remaining quantity: %w", err)
		}
		remaining -= consume
		lastBatchID = batch.ID
	}

	if remaining > 0 {
		return consumedBatchResult{}, fmt.Errorf("insufficient batch quantity to reverse purchase order stock for product %s", productID)
	}

	return consumedBatchResult{batchID: lastBatchID}, nil
}

type inventoryMovementInput struct {
	BusinessID         string
	PurchaseOrderID    string
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

func insertStockMovementTx(ctx context.Context, tx purchaseOrderInventoryTx, req inventoryMovementInput) error {
	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.PurchaseOrderID = strings.TrimSpace(req.PurchaseOrderID)
	req.ReferenceNumber = strings.TrimSpace(req.ReferenceNumber)
	req.LocationID = strings.TrimSpace(req.LocationID)
	req.ProductID = strings.TrimSpace(req.ProductID)
	req.InventoryBalanceID = strings.TrimSpace(req.InventoryBalanceID)
	req.InventoryBatchID = strings.TrimSpace(req.InventoryBatchID)
	req.MovementType = strings.TrimSpace(req.MovementType)
	req.Note = strings.TrimSpace(req.Note)
	req.PerformedBy = strings.TrimSpace(req.PerformedBy)

	if req.BusinessID == "" || req.PurchaseOrderID == "" || req.LocationID == "" || req.ProductID == "" || req.MovementType == "" {
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
			'purchase_order',
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
	`, req.BusinessID, req.ProductID, req.LocationID, req.InventoryBalanceID, req.InventoryBatchID, req.MovementType, req.PurchaseOrderID, req.ReferenceNumber, req.QuantityIn, req.QuantityOut, req.UnitCost, req.StockBefore, req.StockAfter, req.Note, req.PerformedBy)
	if err != nil {
		return fmt.Errorf("insert stock movement: %w", err)
	}

	return nil
}
