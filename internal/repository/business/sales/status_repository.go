package sales

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type salesOrderLifecycleSnapshot struct {
	ID                string
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
				BusinessID: req.BusinessID,
				LocationID: current.LocationID,
				Status:     req.Status,
				CreatedBy:  req.CreatedBy,
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
			location_id::text,
			status,
			COALESCE(reserve_order_items, FALSE),
			COALESCE(sale_id::text, '')
		FROM sales_orders
		WHERE business_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
		LIMIT 1
	`, businessID, salesOrderID).Scan(&snapshot.ID, &snapshot.LocationID, &snapshot.Status, &snapshot.ReserveOrderItems, &snapshot.SaleID); err != nil {
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
