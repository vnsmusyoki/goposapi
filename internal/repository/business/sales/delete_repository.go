package sales

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type salesOrderDeleteSnapshot struct {
	ID                string
	ReferenceNumber   string
	Status            string
	ReserveOrderItems bool
	SaleID            string
}

func DeleteSalesOrderRepository(pool *pgxpool.Pool, businessID, salesOrderID, deletedBy, deletedByName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	salesOrderID = strings.TrimSpace(salesOrderID)
	deletedBy = strings.TrimSpace(deletedBy)
	deletedByName = strings.TrimSpace(deletedByName)
	if businessID == "" || salesOrderID == "" {
		return ErrBusinessNotResolved
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin sales order delete tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err := loadSalesOrderDeleteSnapshotTx(ctx, tx, businessID, salesOrderID)
	if err != nil {
		return err
	}

	if current.SaleID != "" || saleStatusConsumesInventory(current.Status) {
		return ErrSalesOrderCannotDelete
	}

	if current.ReserveOrderItems || strings.EqualFold(current.Status, "approved") || strings.EqualFold(current.Status, "processing") {
		if err := releaseSalesOrderReservationTx(ctx, tx, businessID, salesOrderID); err != nil {
			return err
		}
	}

	if err := CreateSalesOrderLogTx(ctx, tx, SalesOrderLogInput{
		BusinessID:   businessID,
		SalesOrderID: salesOrderID,
		Action:       "deleted",
		ActionedBy:   deletedBy,
		Note:         buildSalesOrderActivityNote("deleted", current.ReferenceNumber, deletedByName, current.Status, "", false, false),
	}); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM sales_order_item_batch_allocations
		WHERE sales_order_id = $1::uuid
	`, salesOrderID); err != nil {
		return fmt.Errorf("delete sales order batch allocations: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE sales_order_items
		SET deleted = TRUE,
		    deleted_at = CURRENT_TIMESTAMP,
		    deleted_by = NULLIF($2, '')::uuid
		WHERE sales_order_id = $1::uuid
		  AND deleted_at IS NULL
	`, salesOrderID, deletedBy); err != nil {
		return fmt.Errorf("soft delete sales order items: %w", err)
	}

	commandTag, err := tx.Exec(ctx, `
		UPDATE sales_orders
		SET deleted = TRUE,
		    deleted_at = CURRENT_TIMESTAMP,
		    deleted_by = NULLIF($3, '')::uuid
		WHERE business_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
	`, businessID, salesOrderID, deletedBy)
	if err != nil {
		return fmt.Errorf("soft delete sales order: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrSaleNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit sales order delete tx: %w", err)
	}

	return nil
}

func loadSalesOrderDeleteSnapshotTx(ctx context.Context, tx saleInventoryTx, businessID, salesOrderID string) (*salesOrderDeleteSnapshot, error) {
	var snapshot salesOrderDeleteSnapshot
	if err := tx.QueryRow(ctx, `
		SELECT
			id::text,
			reference_number,
			status,
			COALESCE(reserve_order_items, FALSE),
			COALESCE(sale_id::text, '')
		FROM sales_orders
		WHERE business_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
		LIMIT 1
	`, businessID, salesOrderID).Scan(&snapshot.ID, &snapshot.ReferenceNumber, &snapshot.Status, &snapshot.ReserveOrderItems, &snapshot.SaleID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSaleNotFound
		}
		return nil, fmt.Errorf("load sales order delete snapshot: %w", err)
	}

	return &snapshot, nil
}
