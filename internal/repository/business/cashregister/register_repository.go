package cashregister

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func OpenCashRegisterRepository(pool *pgxpool.Pool, req OpenRegisterInput) (*ActiveRegister, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.BusinessLocationID = strings.TrimSpace(req.BusinessLocationID)
	req.OpenedBy = strings.TrimSpace(req.OpenedBy)
	req.Notes = strings.TrimSpace(req.Notes)
	if req.BusinessID == "" || req.BusinessLocationID == "" || req.OpenedBy == "" || req.OpeningCashAmount < 0 {
		return nil, ErrInvalidRegisterInput
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin open cash register tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var locationExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM business_locations
			WHERE business_id = $1::uuid
			  AND id::text = $2
		)
	`, req.BusinessID, req.BusinessLocationID).Scan(&locationExists); err != nil {
		return nil, fmt.Errorf("validate cash register location: %w", err)
	}
	if !locationExists {
		return nil, ErrLocationNotFound
	}

	var activeExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM cash_registers
			WHERE business_id = $1::uuid
			  AND business_location_id = $2::uuid
			  AND status = 'open'
		)
	`, req.BusinessID, req.BusinessLocationID).Scan(&activeExists); err != nil {
		return nil, fmt.Errorf("check active cash register: %w", err)
	}
	if activeExists {
		return nil, ErrActiveRegisterExists
	}

	registerNumber, err := nextRegisterNumber(ctx, tx, req.BusinessID)
	if err != nil {
		return nil, err
	}

	var register ActiveRegister
	if err := tx.QueryRow(ctx, `
		INSERT INTO cash_registers (
			business_id,
			business_location_id,
			register_number,
			status,
			opened_by,
			opening_cash_amount,
			expected_closing_cash_amount,
			notes
		)
		VALUES ($1::uuid, $2::uuid, $3, 'open', NULLIF($4, '')::uuid, $5, $5, $6)
		RETURNING
			id::text,
			register_number,
			business_location_id::text,
			status,
			opened_by::text,
			opened_at::text,
			opening_cash_amount,
			expected_closing_cash_amount
	`, req.BusinessID, req.BusinessLocationID, registerNumber, req.OpenedBy, req.OpeningCashAmount, req.Notes).Scan(
		&register.ID,
		&register.RegisterNumber,
		&register.BusinessLocationID,
		&register.Status,
		&register.OpenedBy,
		&register.OpenedAt,
		&register.OpeningCashAmount,
		&register.ExpectedClosingCashAmount,
	); err != nil {
		return nil, fmt.Errorf("open cash register: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO cash_register_transactions (
			business_id,
			cash_register_id,
			transaction_type,
			payment_method,
			amount,
			reference_number,
			notes,
			created_by
		)
		VALUES ($1::uuid, $2::uuid, 'opening_cash', 'cash', $3, $4, $5, NULLIF($6, '')::uuid)
	`, req.BusinessID, register.ID, req.OpeningCashAmount, register.RegisterNumber, "Opening cash", req.OpenedBy); err != nil {
		return nil, fmt.Errorf("record opening cash transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit open cash register tx: %w", err)
	}

	return &register, nil
}

type registerSequenceTx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func nextRegisterNumber(ctx context.Context, tx registerSequenceTx, businessID string) (string, error) {
	var count int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) + 1
		FROM cash_registers
		WHERE business_id = $1::uuid
	`, businessID).Scan(&count); err != nil {
		return "", fmt.Errorf("generate cash register number: %w", err)
	}
	return fmt.Sprintf("CR-%05d", count), nil
}
