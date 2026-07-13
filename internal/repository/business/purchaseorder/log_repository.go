package purchaseorder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4/pgxpool"
)

func CreatePurchaseOrderLogRepository(pool *pgxpool.Pool, req CreatePurchaseOrderLogInput) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return insertPurchaseOrderLog(ctx, pool, req)
}

func CreatePurchaseOrderLogTx(ctx context.Context, tx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, req CreatePurchaseOrderLogInput) error {
	return insertPurchaseOrderLog(ctx, tx, req)
}

func insertPurchaseOrderLog(ctx context.Context, querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, req CreatePurchaseOrderLogInput) error {
	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.PurchaseOrderID = strings.TrimSpace(req.PurchaseOrderID)
	req.Action = strings.TrimSpace(req.Action)
	req.ActionedBy = strings.TrimSpace(req.ActionedBy)
	req.Note = strings.TrimSpace(req.Note)

	if req.BusinessID == "" || req.PurchaseOrderID == "" || req.Action == "" {
		return ErrBusinessNotResolved
	}

	if _, err := querier.Exec(ctx, `
		INSERT INTO purchase_order_logs (
			business_id,
			purchase_order_id,
			action,
			actioned_by,
			note
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			$3,
			NULLIF($4, '')::uuid,
			$5
		)
	`, req.BusinessID, req.PurchaseOrderID, req.Action, req.ActionedBy, req.Note); err != nil {
		return fmt.Errorf("insert purchase order log: %w", err)
	}

	return nil
}
