package settings

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/models"
)

func GetBusinessContactSettingsRepository(pool *pgxpool.Pool, businessID string) (*models.BusinessContactSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	var settings models.BusinessContactSettings
	var creditLimit sql.NullFloat64

	err := pool.QueryRow(ctx, `
		SELECT
			id::text,
			default_credit_limit
		FROM businesses
		WHERE id = $1
		LIMIT 1
	`, businessID).Scan(&settings.ID, &creditLimit)
	if err != nil {
		return nil, fmt.Errorf("load business contact settings: %w", err)
	}

	if creditLimit.Valid {
		value := creditLimit.Float64
		settings.DefaultCreditLimit = &value
	}

	return &settings, nil
}

func UpdateBusinessContactSettingsRepository(pool *pgxpool.Pool, req UpdateBusinessContactSettingsInput) (*models.BusinessContactSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	if req.BusinessID == "" {
		return nil, ErrInvalidBusinessSettingsInput
	}

	_, err := pool.Exec(ctx, `
		UPDATE businesses
		SET default_credit_limit = $2
		WHERE id = $1
	`, req.BusinessID, req.DefaultCreditLimit)
	if err != nil {
		return nil, fmt.Errorf("update business contact settings: %w", err)
	}

	return GetBusinessContactSettingsRepository(pool, req.BusinessID)
}
