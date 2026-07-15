package purchaseorder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4/pgxpool"
)

func CreatePurchaseReturnLogRepository(pool *pgxpool.Pool, req CreatePurchaseReturnLogInput) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return insertPurchaseReturnLog(ctx, pool, req)
}

func CreatePurchaseReturnLogTx(ctx context.Context, tx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, req CreatePurchaseReturnLogInput) error {
	return insertPurchaseReturnLog(ctx, tx, req)
}

func insertPurchaseReturnLog(ctx context.Context, querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, req CreatePurchaseReturnLogInput) error {
	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.PurchaseReturnID = strings.TrimSpace(req.PurchaseReturnID)
	req.Action = strings.TrimSpace(req.Action)
	req.ActionedBy = strings.TrimSpace(req.ActionedBy)
	req.Note = strings.TrimSpace(req.Note)

	if req.BusinessID == "" || req.PurchaseReturnID == "" || req.Action == "" {
		return ErrBusinessNotResolved
	}

	if _, err := querier.Exec(ctx, `
		INSERT INTO purchase_returns_logs (
			business_id,
			purchase_return_id,
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
	`, req.BusinessID, req.PurchaseReturnID, req.Action, req.ActionedBy, req.Note); err != nil {
		return fmt.Errorf("insert purchase return log: %w", err)
	}

	return nil
}
