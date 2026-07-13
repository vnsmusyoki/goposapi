package purchaseorder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func ListPurchaseOrderStatusesRepository(pool *pgxpool.Pool, businessID string) ([]PurchaseOrderStatusDefinition, error) {
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
			editable_note,
			stock_affected_note,
			requires_receiving_items,
			can_be_deleted,
			prepare_invoice,
			sort_order
		FROM purchase_order_statuses
		ORDER BY sort_order ASC, code ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("load purchase order statuses: %w", err)
	}
	defer rows.Close()

	statuses := make([]PurchaseOrderStatusDefinition, 0)
	for rows.Next() {
		var status PurchaseOrderStatusDefinition
		if err := rows.Scan(
			&status.Code,
			&status.Name,
			&status.WhatHappens,
			&status.EditableNote,
			&status.StockAffectedNote,
			&status.RequiresReceivingItems,
			&status.CanBeDeleted,
			&status.PrepareInvoice,
			&status.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("scan purchase order status: %w", err)
		}
		statuses = append(statuses, status)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purchase order statuses: %w", err)
	}

	return statuses, nil
}

func getPurchaseOrderStatusByCode(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, code string) (*PurchaseOrderStatusDefinition, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, nil
	}

	var status PurchaseOrderStatusDefinition
	if err := querier.QueryRow(ctx, `
		SELECT
			code,
			name,
			what_happens,
			editable_note,
			stock_affected_note,
			requires_receiving_items,
			can_be_deleted,
			prepare_invoice,
			sort_order
		FROM purchase_order_statuses
		WHERE code = $1
		LIMIT 1
	`, code).Scan(
		&status.Code,
		&status.Name,
		&status.WhatHappens,
		&status.EditableNote,
		&status.StockAffectedNote,
		&status.RequiresReceivingItems,
		&status.CanBeDeleted,
		&status.PrepareInvoice,
		&status.SortOrder,
	); err != nil {
		return nil, err
	}

	return &status, nil
}
