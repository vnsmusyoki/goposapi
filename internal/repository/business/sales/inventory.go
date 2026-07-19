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
		SELECT
			ib.id::text,
			GREATEST(
				COALESCE(ib.quantity_remaining, 0) - COALESCE(reserved.reserved_quantity, 0),
				0
			)
		FROM inventory_batches ib
		LEFT JOIN (
			SELECT
				inventory_batch_id,
				SUM(allocated_quantity) AS reserved_quantity
			FROM sales_order_item_batch_allocations
			WHERE is_reserved = TRUE
			GROUP BY inventory_batch_id
		) reserved ON reserved.inventory_batch_id = ib.id
		WHERE business_id = $1::uuid
		  AND product_id = $2::uuid
		  AND location_id = $3::uuid
		  AND GREATEST(
				COALESCE(ib.quantity_remaining, 0) - COALESCE(reserved.reserved_quantity, 0),
				0
		  ) > 0
		ORDER BY %s
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
	input.SourceType = strings.TrimSpace(input.SourceType)
	input.ReferenceNumber = strings.TrimSpace(input.ReferenceNumber)
	input.Note = strings.TrimSpace(input.Note)
	input.PerformedBy = strings.TrimSpace(input.PerformedBy)

	if input.BusinessID == "" || input.SaleID == "" || input.LocationID == "" || input.ProductID == "" {
		return nil
	}
	if input.SourceType == "" {
		input.SourceType = "sale"
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
			$7,
			$8::uuid,
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			$15,
			NULLIF($16, '')::uuid,
			CURRENT_TIMESTAMP,
			CURRENT_TIMESTAMP
		)
	`, input.BusinessID, input.ProductID, input.LocationID, input.InventoryBalanceID, input.InventoryBatchID, input.MovementType, input.SourceType, input.SaleID, input.ReferenceNumber, input.QuantityIn, input.QuantityOut, input.UnitCost, input.StockBefore, input.StockAfter, input.Note, input.PerformedBy)
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
	SourceType         string
	MovementType       string
	QuantityIn         float64
	QuantityOut        float64
	UnitCost           float64
	StockBefore        float64
	StockAfter         float64
	Note               string
	PerformedBy        string
}

func applySalesOrderInventoryAllocationTx(
	ctx context.Context,
	tx saleInventoryTx,
	req CreateSaleOrderInput,
	item CreateSaleItemInput,
	salesOrderID, salesOrderItemID, productName, sku, unit string,
	isReserved bool,
) error {
	if item.Quantity <= 0 {
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
	if available < item.Quantity {
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

		if _, err := tx.Exec(ctx, `
			INSERT INTO sales_order_item_batch_allocations (
				sales_order_id,
				sales_order_item_id,
				business_id,
				inventory_batch_id,
				allocated_quantity,
				unit_cost,
				line_total,
				sort_order,
				is_reserved
			)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8, $9)
		`, salesOrderID, salesOrderItemID, req.BusinessID, batch.ID, allocate, item.UnitCost, allocate*item.UnitPrice, idx, isReserved); err != nil {
			return fmt.Errorf("insert sales order batch allocation: %w", err)
		}

		if isReserved {
			nextAvailable := balance.QuantityAvailable - allocate
			nextReserved := balance.QuantityReserved + allocate
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
			balance.QuantityAvailable = nextAvailable
			balance.QuantityReserved = nextReserved
		}

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

type salesOrderItemSnapshot struct {
	ID                   string
	ProductID            string
	ProductName          string
	SKU                  string
	Unit                 string
	Quantity             float64
	UnitCost             float64
	DiscountPercentage   float64
	DiscountAmount       float64
	TaxRate              float64
	TaxAmount            float64
	UnitPrice            float64
	LineTotal            float64
	BatchTrackingEnabled bool
	SortOrder            int
}

type salesOrderBatchSnapshot struct {
	ItemID     string
	BatchID    string
	Quantity   float64
	UnitCost   float64
	LineTotal  float64
	SortOrder  int
	IsReserved bool
}

func finalizeSalesOrderInventoryTx(
	ctx context.Context,
	tx saleInventoryTx,
	req CreateSaleOrderInput,
	salesOrderID string,
	saleID string,
	createdBy string,
) error {
	var existingSaleID string
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(sale_id::text, '')
		FROM sales_orders
		WHERE id = $1::uuid
		FOR UPDATE
	`, salesOrderID).Scan(&existingSaleID); err != nil {
		return fmt.Errorf("check sales order link: %w", err)
	}
	if strings.TrimSpace(existingSaleID) != "" {
		return nil
	}

	var referenceNumber string
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(reference_number, '')
		FROM sales_orders
		WHERE id = $1::uuid
		LIMIT 1
	`, salesOrderID).Scan(&referenceNumber); err != nil {
		return fmt.Errorf("load sales order header: %w", err)
	}

	var insertedSaleID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO sales (
			business_id,
			location_id,
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
			created_by
		)
		SELECT
			business_id,
			location_id,
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
			NULLIF($2, '')::uuid
		FROM sales_orders
		WHERE id = $1::uuid
		RETURNING id::text
	`, salesOrderID, createdBy).Scan(&insertedSaleID); err != nil {
		return fmt.Errorf("insert finalized sale: %w", err)
	}
	if saleID != "" && saleID != insertedSaleID {
		saleID = insertedSaleID
	}
	if saleID == "" {
		saleID = insertedSaleID
	}

	itemRows, err := tx.Query(ctx, `
		SELECT
			id::text,
			product_id::text,
			product_name,
			COALESCE(sku, ''),
			COALESCE(unit, ''),
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
		FROM sales_order_items
		WHERE sales_order_id = $1::uuid
		ORDER BY sort_order ASC, created_at ASC, id ASC
	`, salesOrderID)
	if err != nil {
		return fmt.Errorf("load sales order items: %w", err)
	}
	defer itemRows.Close()

	items := make([]salesOrderItemSnapshot, 0)
	for itemRows.Next() {
		var item salesOrderItemSnapshot
		if err := itemRows.Scan(&item.ID, &item.ProductID, &item.ProductName, &item.SKU, &item.Unit, &item.Quantity, &item.UnitCost, &item.DiscountPercentage, &item.DiscountAmount, &item.TaxRate, &item.TaxAmount, &item.UnitPrice, &item.LineTotal, &item.BatchTrackingEnabled, &item.SortOrder); err != nil {
			return fmt.Errorf("scan sales order item: %w", err)
		}
		items = append(items, item)
	}
	if err := itemRows.Err(); err != nil {
		return fmt.Errorf("iterate sales order items: %w", err)
	}

	saleItemIDs := map[string]string{}
	for _, item := range items {
		var saleItemID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO sale_items (
				sale_id,
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
		`, saleID, req.BusinessID, item.ProductID, item.ProductName, item.SKU, item.Unit, item.Quantity, item.UnitCost, item.DiscountPercentage, item.DiscountAmount, item.TaxRate, item.TaxAmount, item.UnitPrice, item.LineTotal, item.BatchTrackingEnabled, item.SortOrder).Scan(&saleItemID); err != nil {
			return fmt.Errorf("insert sale item: %w", err)
		}
		saleItemIDs[item.ID] = saleItemID
	}

	allocationRows, err := tx.Query(ctx, `
		SELECT
			sales_order_item_id::text,
			inventory_batch_id::text,
			allocated_quantity,
			unit_cost,
			line_total,
			sort_order,
			is_reserved
		FROM sales_order_item_batch_allocations
		WHERE sales_order_id = $1::uuid
		ORDER BY sort_order ASC, created_at ASC, id ASC
	`, salesOrderID)
	if err != nil {
		return fmt.Errorf("load sales order batch allocations: %w", err)
	}
	defer allocationRows.Close()

	balanceByProduct := map[string]*saleInventoryBalanceSnapshot{}
	reservedByProduct := map[string]float64{}
	consumedByProduct := map[string]float64{}

	for allocationRows.Next() {
		var (
			orderItemID string
			batchID     string
			quantity    float64
			unitCost    float64
			lineTotal   float64
			sortOrder   int
			isReserved  bool
		)
		if err := allocationRows.Scan(&orderItemID, &batchID, &quantity, &unitCost, &lineTotal, &sortOrder, &isReserved); err != nil {
			return fmt.Errorf("scan sales order batch allocation: %w", err)
		}

		var productID string
		if err := tx.QueryRow(ctx, `
			SELECT product_id::text
			FROM sales_order_items
			WHERE id = $1::uuid
			LIMIT 1
		`, orderItemID).Scan(&productID); err != nil {
			return fmt.Errorf("load sales order item product: %w", err)
		}

		balance, ok := balanceByProduct[productID]
		if !ok {
			b, err := getOrCreateSaleInventoryBalanceTx(ctx, tx, req.BusinessID, productID, req.LocationID)
			if err != nil {
				return err
			}
			balanceByProduct[productID] = &b
			balance = &b
		}

		var currentRemaining float64
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(quantity_remaining, 0)
			FROM inventory_batches
			WHERE id = $1::uuid
			FOR UPDATE
		`, batchID).Scan(&currentRemaining); err != nil {
			return fmt.Errorf("load inventory batch: %w", err)
		}
		if currentRemaining < quantity {
			return fmt.Errorf("insufficient stock for product %s", productID)
		}

		if _, err := tx.Exec(ctx, `
			UPDATE inventory_batches
			SET quantity_remaining = quantity_remaining - $2,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $1::uuid
		`, batchID, quantity); err != nil {
			return fmt.Errorf("update inventory batch: %w", err)
		}

		saleItemID := saleItemIDs[orderItemID]
		if saleItemID == "" {
			return fmt.Errorf("sale item not found for sales order item %s", orderItemID)
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
		`, saleID, saleItemID, req.BusinessID, batchID, quantity, unitCost, lineTotal, sortOrder); err != nil {
			return fmt.Errorf("insert sale batch allocation: %w", err)
		}

		if err := insertSaleStockMovementTx(ctx, tx, saleStockMovementInput{
			BusinessID:         req.BusinessID,
			SaleID:             saleID,
			ReferenceNumber:    referenceNumber,
			LocationID:         req.LocationID,
			ProductID:          productID,
			InventoryBalanceID: balance.ID,
			InventoryBatchID:   batchID,
			SourceType:         "sale",
			MovementType:       "sale",
			QuantityIn:         0,
			QuantityOut:        quantity,
			UnitCost:           unitCost,
			StockBefore:        balance.QuantityAvailable + balance.QuantityReserved,
			StockAfter:         balance.QuantityAvailable + balance.QuantityReserved - quantity,
			Note:               "Recorded finalized sale deduction.",
			PerformedBy:        createdBy,
		}); err != nil {
			return err
		}

		if isReserved {
			reservedByProduct[productID] += quantity
		} else {
			consumedByProduct[productID] += quantity
		}
	}
	if err := allocationRows.Err(); err != nil {
		return fmt.Errorf("iterate sales order batch allocations: %w", err)
	}

	for productID, consumedQty := range consumedByProduct {
		if consumedQty <= 0 {
			continue
		}
		balance, ok := balanceByProduct[productID]
		if !ok {
			b, err := getOrCreateSaleInventoryBalanceTx(ctx, tx, req.BusinessID, productID, req.LocationID)
			if err != nil {
				return err
			}
			balanceByProduct[productID] = &b
			balance = &b
		}
		nextAvailable := balance.QuantityAvailable - consumedQty
		if nextAvailable < 0 {
			nextAvailable = 0
		}
		if _, err := tx.Exec(ctx, `
			UPDATE inventory_balances
			SET quantity_available = $2,
			    last_movement_at = CURRENT_TIMESTAMP,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $1::uuid
		`, balance.ID, nextAvailable); err != nil {
			return fmt.Errorf("update inventory balance availability: %w", err)
		}
		balance.QuantityAvailable = nextAvailable
	}

	for productID, reservedQty := range reservedByProduct {
		if reservedQty <= 0 {
			continue
		}
		balance, ok := balanceByProduct[productID]
		if !ok {
			b, err := getOrCreateSaleInventoryBalanceTx(ctx, tx, req.BusinessID, productID, req.LocationID)
			if err != nil {
				return err
			}
			balanceByProduct[productID] = &b
			balance = &b
		}
		nextReserved := balance.QuantityReserved - reservedQty
		if nextReserved < 0 {
			nextReserved = 0
		}
		if _, err := tx.Exec(ctx, `
			UPDATE inventory_balances
			SET quantity_reserved = $2,
			    last_movement_at = CURRENT_TIMESTAMP,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $1::uuid
		`, balance.ID, nextReserved); err != nil {
			return fmt.Errorf("release inventory reservation: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE sales_order_item_batch_allocations
		SET is_reserved = FALSE,
		    updated_at = CURRENT_TIMESTAMP
		WHERE sales_order_id = $1::uuid
		  AND is_reserved = TRUE
	`, salesOrderID); err != nil {
		return fmt.Errorf("release sales order allocations: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE sales_orders
		SET sale_id = $2::uuid,
		    converted_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1::uuid
	`, salesOrderID, saleID); err != nil {
		return fmt.Errorf("link sales order to sale: %w", err)
	}

	return nil
}
