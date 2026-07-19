package cashregister

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func GetPosReadinessRepository(pool *pgxpool.Pool, businessID, userID, businessLocationID string) (*PosReadiness, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	userID = strings.TrimSpace(userID)
	businessLocationID = strings.TrimSpace(businessLocationID)
	if businessID == "" || userID == "" {
		return nil, ErrBusinessNotResolved
	}

	location, err := resolveReadinessLocation(ctx, pool, businessID, businessLocationID)
	if err != nil {
		return nil, err
	}

	readiness := &PosReadiness{
		BusinessLocationID:   location.ID,
		BusinessLocationName: location.Name,
		PaymentMethods:       location.PaymentMethods,
		PrinterConfigured:    true,
		PrinterTestRequired:  true,
		MpesaConfigured:      location.hasPaymentMethod("mpesa"),
		MpesaStkPushEnabled:  location.hasPaymentMethod("mpesa"),
		Warnings:             []string{"Printer test has not been completed for this POS session."},
	}

	activeRegister, err := getActiveCashRegisterForUser(ctx, pool, businessID, userID, location.ID)
	if err != nil {
		return nil, err
	}
	if activeRegister == nil {
		readiness.BlockingReasons = append(readiness.BlockingReasons, "Open a cash register for this location before using POS.")
	} else {
		readiness.HasActiveCashRegister = true
		readiness.ActiveRegister = activeRegister
	}

	if !readiness.MpesaConfigured {
		readiness.Warnings = append(readiness.Warnings, "MPesa is not configured for this location. STK Push will be disabled.")
	}

	return readiness, nil
}

type readinessLocation struct {
	ID             string
	Name           string
	PaymentMethods []string
}

func (location readinessLocation) hasPaymentMethod(method string) bool {
	target := strings.ToLower(strings.TrimSpace(method))
	for _, value := range location.PaymentMethods {
		normalized := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(value)))
		if normalized == target || normalized == "mpesa" || normalized == "mpesastk" || normalized == "stkpush" {
			return true
		}
	}
	return false
}

func resolveReadinessLocation(ctx context.Context, pool *pgxpool.Pool, businessID, businessLocationID string) (readinessLocation, error) {
	query := `
		SELECT
			id::text,
			location_name,
			COALESCE(payment_methods::text, '[]')
		FROM business_locations
		WHERE business_id = $1::uuid
	`
	args := []any{businessID}
	if businessLocationID != "" {
		query += " AND id::text = $2"
		args = append(args, businessLocationID)
	}
	query += " ORDER BY created_at ASC LIMIT 1"

	var location readinessLocation
	var rawPaymentMethods string
	if err := pool.QueryRow(ctx, query, args...).Scan(&location.ID, &location.Name, &rawPaymentMethods); err != nil {
		if err == pgx.ErrNoRows {
			return readinessLocation{}, ErrLocationNotFound
		}
		return readinessLocation{}, fmt.Errorf("load POS readiness location: %w", err)
	}
	if err := json.Unmarshal([]byte(rawPaymentMethods), &location.PaymentMethods); err != nil {
		location.PaymentMethods = []string{"cash"}
	}
	return location, nil
}

func getActiveCashRegisterForUser(ctx context.Context, pool *pgxpool.Pool, businessID, userID, businessLocationID string) (*ActiveRegister, error) {
	var register ActiveRegister
	var openedBy sql.NullString
	if err := pool.QueryRow(ctx, `
		SELECT
			id::text,
			COALESCE(register_number, ''),
			business_location_id::text,
			status,
			opened_by::text,
			opened_at::text,
			opening_cash_amount,
			expected_closing_cash_amount
		FROM cash_registers
		WHERE business_id = $1::uuid
		  AND business_location_id = $2::uuid
		  AND opened_by = $3::uuid
		  AND status = 'open'
		ORDER BY opened_at DESC
		LIMIT 1
	`, businessID, businessLocationID, userID).Scan(
		&register.ID,
		&register.RegisterNumber,
		&register.BusinessLocationID,
		&register.Status,
		&openedBy,
		&register.OpenedAt,
		&register.OpeningCashAmount,
		&register.ExpectedClosingCashAmount,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("load active cash register: %w", err)
	}
	register.OpenedBy = openedBy.String
	return &register, nil
}
