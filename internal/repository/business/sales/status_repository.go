package sales

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func ListSalesOrderStatusesRepository(pool *pgxpool.Pool, businessID string) ([]SalesOrderStatusDefinition, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	rows, err := pool.Query(ctx, `
		SELECT
			code,
			name,
			what_happens,
			requires_further_action,
			sort_order
		FROM sale_order_statuses
		ORDER BY sort_order ASC, code ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("load sales order statuses: %w", err)
	}
	defer rows.Close()

	statuses := make([]SalesOrderStatusDefinition, 0)
	for rows.Next() {
		var status SalesOrderStatusDefinition
		if err := rows.Scan(
			&status.Code,
			&status.Name,
			&status.WhatHappens,
			&status.RequiresFurtherAction,
			&status.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("scan sales order status: %w", err)
		}
		statuses = append(statuses, status)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sales order statuses: %w", err)
	}

	return statuses, nil
}

func getSalesOrderStatusByCode(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, code string) (*SalesOrderStatusDefinition, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrSalesOrderStatusDefinitionNotFound
	}

	var status SalesOrderStatusDefinition
	if err := querier.QueryRow(ctx, `
		SELECT
			code,
			name,
			what_happens,
			requires_further_action,
			sort_order
		FROM sale_order_statuses
		WHERE code = $1
		LIMIT 1
	`, code).Scan(
		&status.Code,
		&status.Name,
		&status.WhatHappens,
		&status.RequiresFurtherAction,
		&status.SortOrder,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSalesOrderStatusDefinitionNotFound
		}
		return nil, fmt.Errorf("load sales order status definition: %w", err)
	}

	return &status, nil
}

func resolveSalesOrderStatusDefinition(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, code string) (*SalesOrderStatusDefinition, error) {
	return getSalesOrderStatusByCode(ctx, querier, code)
}

func loadSalesOrderStatusTransitionTx(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, currentStatus, nextStatus string) (*SalesOrderStatusDefinition, *SalesOrderStatusDefinition, error) {
	current, err := getSalesOrderStatusByCode(ctx, querier, currentStatus)
	if err != nil {
		return nil, nil, err
	}

	next, err := getSalesOrderStatusByCode(ctx, querier, nextStatus)
	if err != nil {
		return nil, nil, err
	}

	if current != nil && next != nil && next.SortOrder < current.SortOrder {
		return nil, nil, ErrSalesOrderStatusRegressionNotAllowed
	}

	return current, next, nil
}

type salesOrderLifecycleSnapshot struct {
	ID                string
	ReferenceNumber   string
	LocationID        string
	Status            string
	ReserveOrderItems bool
	SaleID            string
}

func UpdateSalesOrderStatusRepository(pool *pgxpool.Pool, req UpdateSalesOrderStatusInput) (*Sale, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.SalesOrderID = strings.TrimSpace(req.SalesOrderID)
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)

	if req.BusinessID == "" || req.SalesOrderID == "" || req.Status == "" {
		return nil, ErrInvalidSaleInput
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin sales order status tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err := loadSalesOrderLifecycleSnapshotTx(ctx, tx, req.BusinessID, req.SalesOrderID)
	if err != nil {
		return nil, err
	}

	_, _, err = loadSalesOrderStatusTransitionTx(ctx, tx, current.Status, req.Status)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE sales_orders
		SET status = $2,
		    reserve_order_items = $3,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1::uuid
	`, req.SalesOrderID, req.Status, req.ReserveOrderItems); err != nil {
		return nil, fmt.Errorf("update sales order status: %w", err)
	}

	if saleStatusConsumesInventory(req.Status) {
		if strings.TrimSpace(current.SaleID) == "" {
			if err := finalizeSalesOrderInventoryTx(ctx, tx, CreateSaleOrderInput{
				BusinessID:    req.BusinessID,
				LocationID:    current.LocationID,
				Status:        req.Status,
				CreatedBy:     req.CreatedBy,
				CreatedByName: req.CreatedByName,
			}, req.SalesOrderID, "", req.CreatedBy); err != nil {
				return nil, err
			}
		}
	} else if req.ReserveOrderItems && (req.Status == "approved" || req.Status == "processing") {
		if err := reserveSalesOrderInventoryTx(ctx, tx, req.BusinessID, req.SalesOrderID); err != nil {
			return nil, err
		}
	} else if current.ReserveOrderItems && !req.ReserveOrderItems {
		if err := releaseSalesOrderReservationTx(ctx, tx, req.BusinessID, req.SalesOrderID); err != nil {
			return nil, err
		}
	}

	logAction := "status_changed"
	if saleStatusConsumesInventory(req.Status) && strings.TrimSpace(current.SaleID) == "" {
		logAction = "finalized"
	}
	if err := CreateSalesOrderLogTx(ctx, tx, SalesOrderLogInput{
		BusinessID:   req.BusinessID,
		SalesOrderID: req.SalesOrderID,
		Action:       logAction,
		ActionedBy:   req.CreatedBy,
		Note:         buildSalesOrderActivityNote(logAction, current.ReferenceNumber, req.CreatedByName, current.Status, req.Status, req.ReserveOrderItems, logAction == "finalized"),
	}); err != nil {
		return nil, err
	}

	updated, err := GetSaleByIDRepositoryTx(ctx, tx, req.BusinessID, req.SalesOrderID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit sales order status tx: %w", err)
	}

	return updated, nil
}

func loadSalesOrderLifecycleSnapshotTx(ctx context.Context, tx saleInventoryTx, businessID, salesOrderID string) (*salesOrderLifecycleSnapshot, error) {
	var snapshot salesOrderLifecycleSnapshot
	if err := tx.QueryRow(ctx, `
		SELECT
			id::text,
			reference_number,
			location_id::text,
			status,
			COALESCE(reserve_order_items, FALSE),
			COALESCE(sale_id::text, '')
		FROM sales_orders
		WHERE business_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
		LIMIT 1
	`, businessID, salesOrderID).Scan(&snapshot.ID, &snapshot.ReferenceNumber, &snapshot.LocationID, &snapshot.Status, &snapshot.ReserveOrderItems, &snapshot.SaleID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSaleNotFound
		}
		return nil, fmt.Errorf("load sales order lifecycle snapshot: %w", err)
	}

	return &snapshot, nil
}

func reserveSalesOrderInventoryTx(ctx context.Context, tx saleInventoryTx, businessID, salesOrderID string) error {
	var locationID string
	if err := tx.QueryRow(ctx, `
		SELECT location_id::text
		FROM sales_orders
		WHERE id = $1::uuid
		LIMIT 1
	`, salesOrderID).Scan(&locationID); err != nil {
		return fmt.Errorf("load sales order location: %w", err)
	}

	rows, err := tx.Query(ctx, `
		SELECT
			soi.product_id::text,
			SUM(soia.allocated_quantity)
		FROM sales_order_item_batch_allocations soia
		INNER JOIN sales_order_items soi ON soi.id = soia.sales_order_item_id
		WHERE soia.sales_order_id = $1::uuid
		  AND soia.is_reserved = FALSE
		GROUP BY soi.product_id
	`, salesOrderID)
	if err != nil {
		return fmt.Errorf("load sales order reservation allocations: %w", err)
	}
	defer rows.Close()

	type reservationGroup struct {
		productID string
		quantity  float64
	}

	groups := make([]reservationGroup, 0)
	for rows.Next() {
		var group reservationGroup
		if err := rows.Scan(&group.productID, &group.quantity); err != nil {
			return fmt.Errorf("scan sales order reservation allocation: %w", err)
		}
		if group.quantity > 0 {
			groups = append(groups, group)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate sales order reservation allocations: %w", err)
	}

	for _, group := range groups {
		balance, err := getOrCreateSaleInventoryBalanceTx(ctx, tx, businessID, group.productID, locationID)
		if err != nil {
			return err
		}
		if balance.QuantityAvailable < group.quantity {
			return fmt.Errorf("insufficient stock for product %s", group.productID)
		}

		nextAvailable := balance.QuantityAvailable - group.quantity
		nextReserved := balance.QuantityReserved + group.quantity
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
			return fmt.Errorf("update inventory balance for reservation: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE sales_order_item_batch_allocations
		SET is_reserved = TRUE,
		    updated_at = CURRENT_TIMESTAMP
		WHERE sales_order_id = $1::uuid
		  AND is_reserved = FALSE
	`, salesOrderID); err != nil {
		return fmt.Errorf("mark sales order allocations reserved: %w", err)
	}

	return nil
}

func releaseSalesOrderReservationTx(ctx context.Context, tx saleInventoryTx, businessID, salesOrderID string) error {
	rows, err := tx.Query(ctx, `
		SELECT
			soi.product_id::text,
			SUM(soia.allocated_quantity)
		FROM sales_order_item_batch_allocations soia
		INNER JOIN sales_order_items soi ON soi.id = soia.sales_order_item_id
		WHERE soia.sales_order_id = $1::uuid
		  AND soia.is_reserved = TRUE
		GROUP BY soi.product_id
	`, salesOrderID)
	if err != nil {
		return fmt.Errorf("load sales order reservation release allocations: %w", err)
	}
	defer rows.Close()

	type releaseGroup struct {
		productID string
		quantity  float64
	}

	groups := make([]releaseGroup, 0)
	for rows.Next() {
		var group releaseGroup
		if err := rows.Scan(&group.productID, &group.quantity); err != nil {
			return fmt.Errorf("scan sales order reservation release allocation: %w", err)
		}
		if group.quantity > 0 {
			groups = append(groups, group)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate sales order reservation release allocations: %w", err)
	}

	var locationID string
	if err := tx.QueryRow(ctx, `
		SELECT location_id::text
		FROM sales_orders
		WHERE id = $1::uuid
		LIMIT 1
	`, salesOrderID).Scan(&locationID); err != nil {
		return fmt.Errorf("load sales order location: %w", err)
	}

	for _, group := range groups {
		balance, err := getOrCreateSaleInventoryBalanceTx(ctx, tx, businessID, group.productID, locationID)
		if err != nil {
			return err
		}

		nextAvailable := balance.QuantityAvailable + group.quantity
		nextReserved := balance.QuantityReserved - group.quantity
		if nextReserved < 0 {
			nextReserved = 0
		}
		if _, err := tx.Exec(ctx, `
			UPDATE inventory_balances
			SET quantity_available = $2,
			    quantity_reserved = $3,
			    last_movement_at = CURRENT_TIMESTAMP,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $1::uuid
		`, balance.ID, nextAvailable, nextReserved); err != nil {
			return fmt.Errorf("update inventory balance for release: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE sales_order_item_batch_allocations
		SET is_reserved = FALSE,
		    updated_at = CURRENT_TIMESTAMP
		WHERE sales_order_id = $1::uuid
		  AND is_reserved = TRUE
	`, salesOrderID); err != nil {
		return fmt.Errorf("mark sales order allocations released: %w", err)
	}

	return nil
}
