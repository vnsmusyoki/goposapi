package sales

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4/pgxpool"
)

func CreateSalesOrderLogRepository(pool *pgxpool.Pool, req SalesOrderLogInput) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return insertSalesOrderLog(ctx, pool, req)
}

func CreateSalesOrderLogTx(ctx context.Context, tx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, req SalesOrderLogInput) error {
	return insertSalesOrderLog(ctx, tx, req)
}

type SalesOrderLogInput struct {
	BusinessID   string
	SalesOrderID string
	Action       string
	ActionedBy   string
	Note         string
}

func insertSalesOrderLog(ctx context.Context, querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, req SalesOrderLogInput) error {
	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.SalesOrderID = strings.TrimSpace(req.SalesOrderID)
	req.Action = strings.TrimSpace(req.Action)
	req.ActionedBy = strings.TrimSpace(req.ActionedBy)
	req.Note = strings.TrimSpace(req.Note)

	if req.BusinessID == "" || req.SalesOrderID == "" || req.Action == "" {
		return ErrBusinessNotResolved
	}

	if _, err := querier.Exec(ctx, `
		INSERT INTO sales_order_logs (
			business_id,
			sales_order_id,
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
	`, req.BusinessID, req.SalesOrderID, req.Action, req.ActionedBy, req.Note); err != nil {
		return fmt.Errorf("insert sales order log: %w", err)
	}

	return nil
}
