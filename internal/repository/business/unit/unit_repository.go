package unit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/models"
)

func ListBusinessUnitsRepository(pool *pgxpool.Pool, businessID string) ([]models.BusinessUnit, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	rows, err := pool.Query(ctx, `
		SELECT
			id::text,
			business_id::text,
			name,
			short_name,
			allow_decimal,
			is_multiple_of_other,
			COALESCE(base_unit_id::text, ''),
			COALESCE(conversion_rate::float8, 0),
			COALESCE(created_by_user_id::text, ''),
			COALESCE(created_by, ''),
			created_at::text,
			updated_at::text
		FROM business_units
		WHERE business_id = $1
		ORDER BY created_at DESC, name ASC
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list business units: %w", err)
	}
	defer rows.Close()

	units := make([]models.BusinessUnit, 0)
	for rows.Next() {
		var unit models.BusinessUnit
		if err := rows.Scan(
			&unit.ID,
			&unit.BusinessID,
			&unit.Name,
			&unit.ShortName,
			&unit.AllowDecimal,
			&unit.IsMultipleOfOther,
			&unit.BaseUnitID,
			&unit.ConversionRate,
			&unit.CreatedByUserID,
			&unit.CreatedBy,
			&unit.CreatedAt,
			&unit.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan business unit: %w", err)
		}
		units = append(units, unit)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate business units: %w", err)
	}

	return units, nil
}

func CreateBusinessUnitRepository(pool *pgxpool.Pool, req CreateBusinessUnitInput) (*models.BusinessUnit, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.Name = strings.TrimSpace(req.Name)
	req.ShortName = strings.TrimSpace(req.ShortName)
	req.BaseUnitID = strings.TrimSpace(req.BaseUnitID)
	req.CreatedByUserID = strings.TrimSpace(req.CreatedByUserID)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)

	if req.BusinessID == "" || req.Name == "" || req.ShortName == "" {
		return nil, ErrInvalidBusinessUnitInput
	}

	if req.IsMultipleOfOther {
		if req.BaseUnitID == "" || req.ConversionRate <= 0 {
			return nil, ErrInvalidBusinessUnitInput
		}
	} else {
		req.BaseUnitID = ""
		req.ConversionRate = 0
	}

	if exists, err := businessUnitExists(ctx, pool, req.BusinessID, "", req.Name, "name"); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrBusinessUnitAlreadyExists
	}

	if exists, err := businessUnitExists(ctx, pool, req.BusinessID, "", req.ShortName, "short_name"); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrBusinessUnitAlreadyExists
	}

	if req.IsMultipleOfOther {
		validBase, err := businessUnitBaseExists(ctx, pool, req.BusinessID, req.BaseUnitID, "")
		if err != nil {
			return nil, err
		}
		if !validBase {
			return nil, ErrInvalidBusinessUnitInput
		}
	}

	var unit models.BusinessUnit
	err := pool.QueryRow(ctx, `
		INSERT INTO business_units (
			business_id,
			name,
			short_name,
			allow_decimal,
			is_multiple_of_other,
			base_unit_id,
			conversion_rate,
			created_by_user_id,
			created_by
		)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::uuid, $7, NULLIF($8, '')::uuid, $9)
		RETURNING
			id::text,
			business_id::text,
			name,
			short_name,
			allow_decimal,
			is_multiple_of_other,
			COALESCE(base_unit_id::text, ''),
			COALESCE(conversion_rate::float8, 0),
			COALESCE(created_by_user_id::text, ''),
			COALESCE(created_by, ''),
			created_at::text,
			updated_at::text
	`, req.BusinessID, req.Name, req.ShortName, req.AllowDecimal, req.IsMultipleOfOther, req.BaseUnitID, req.ConversionRate, req.CreatedByUserID, req.CreatedBy).Scan(
		&unit.ID,
		&unit.BusinessID,
		&unit.Name,
		&unit.ShortName,
		&unit.AllowDecimal,
		&unit.IsMultipleOfOther,
		&unit.BaseUnitID,
		&unit.ConversionRate,
		&unit.CreatedByUserID,
		&unit.CreatedBy,
		&unit.CreatedAt,
		&unit.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert business unit: %w", err)
	}

	return &unit, nil
}

func UpdateBusinessUnitRepository(pool *pgxpool.Pool, req UpdateBusinessUnitInput) (*models.BusinessUnit, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.ID = strings.TrimSpace(req.ID)
	req.Name = strings.TrimSpace(req.Name)
	req.ShortName = strings.TrimSpace(req.ShortName)
	req.BaseUnitID = strings.TrimSpace(req.BaseUnitID)

	if req.BusinessID == "" || req.ID == "" || req.Name == "" || req.ShortName == "" {
		return nil, ErrInvalidBusinessUnitInput
	}

	if req.IsMultipleOfOther {
		if req.BaseUnitID == "" || req.ConversionRate <= 0 {
			return nil, ErrInvalidBusinessUnitInput
		}
	} else {
		req.BaseUnitID = ""
		req.ConversionRate = 0
	}

	if exists, err := businessUnitExists(ctx, pool, req.BusinessID, req.ID, req.Name, "name"); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrBusinessUnitAlreadyExists
	}

	if exists, err := businessUnitExists(ctx, pool, req.BusinessID, req.ID, req.ShortName, "short_name"); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrBusinessUnitAlreadyExists
	}

	if req.IsMultipleOfOther {
		validBase, err := businessUnitBaseExists(ctx, pool, req.BusinessID, req.BaseUnitID, req.ID)
		if err != nil {
			return nil, err
		}
		if !validBase {
			return nil, ErrInvalidBusinessUnitInput
		}
	}

	var unit models.BusinessUnit
	row := pool.QueryRow(ctx, `
		UPDATE business_units
		SET name = $3,
			short_name = $4,
			allow_decimal = $5,
			is_multiple_of_other = $6,
			base_unit_id = NULLIF($7, '')::uuid,
			conversion_rate = $8
		WHERE business_id = $1
		  AND id::text = $2
		RETURNING
			id::text,
			business_id::text,
			name,
			short_name,
			allow_decimal,
			is_multiple_of_other,
			COALESCE(base_unit_id::text, ''),
			COALESCE(conversion_rate::float8, 0),
			COALESCE(created_by_user_id::text, ''),
			COALESCE(created_by, ''),
			created_at::text,
			updated_at::text
	`, req.BusinessID, req.ID, req.Name, req.ShortName, req.AllowDecimal, req.IsMultipleOfOther, req.BaseUnitID, req.ConversionRate)

	if err := row.Scan(
		&unit.ID,
		&unit.BusinessID,
		&unit.Name,
		&unit.ShortName,
		&unit.AllowDecimal,
		&unit.IsMultipleOfOther,
		&unit.BaseUnitID,
		&unit.ConversionRate,
		&unit.CreatedByUserID,
		&unit.CreatedBy,
		&unit.CreatedAt,
		&unit.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrBusinessUnitNotFound
		}
		return nil, fmt.Errorf("update business unit: %w", err)
	}

	return &unit, nil
}

func DeleteBusinessUnitRepository(pool *pgxpool.Pool, businessID, unitID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	unitID = strings.TrimSpace(unitID)
	if businessID == "" || unitID == "" {
		return ErrInvalidBusinessUnitInput
	}

	result, err := pool.Exec(ctx, `
		DELETE FROM business_units
		WHERE business_id = $1
		  AND id::text = $2
	`, businessID, unitID)
	if err != nil {
		return fmt.Errorf("delete business unit: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrBusinessUnitNotFound
	}

	return nil
}

func businessUnitExists(ctx context.Context, pool *pgxpool.Pool, businessID, excludeID, value, column string) (bool, error) {
	if column != "short_name" {
		column = "name"
	}

	query := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1
			FROM business_units
			WHERE business_id = $1
			  AND LOWER(%s) = LOWER($2)
	`, column)

	args := []any{businessID, value}
	if excludeID != "" {
		query += " AND id::text <> $3"
		args = append(args, excludeID)
	}
	query += ")"

	var exists bool
	if err := pool.QueryRow(ctx, query, args...).Scan(&exists); err != nil {
		return false, fmt.Errorf("check business unit duplicate %s: %w", column, err)
	}

	return exists, nil
}

func businessUnitBaseExists(ctx context.Context, pool *pgxpool.Pool, businessID, baseUnitID, excludeID string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM business_units
			WHERE business_id = $1
			  AND id::text = $2
			  AND is_multiple_of_other = FALSE
	`
	args := []any{businessID, baseUnitID}
	if excludeID != "" {
		query += " AND id::text <> $3"
		args = append(args, excludeID)
	}
	query += ")"

	var exists bool
	if err := pool.QueryRow(ctx, query, args...).Scan(&exists); err != nil {
		return false, fmt.Errorf("check base business unit: %w", err)
	}
	return exists, nil
}
