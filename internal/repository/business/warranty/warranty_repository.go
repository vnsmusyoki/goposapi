package warranty

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/models"
)

func ListWarrantiesRepository(pool *pgxpool.Pool, businessID string) ([]models.Warranty, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	rows, err := pool.Query(ctx, `
		SELECT
			w.id::text,
			w.business_id::text,
			w.name,
			COALESCE(w.description, ''),
			w.duration_value,
			w.duration_unit,
			COALESCE(w.added_by::text, ''),
			COALESCE(u.full_name, ''),
			w.created_at::text,
			w.updated_at::text
		FROM product_warranties w
		LEFT JOIN users u ON u.id = w.added_by
		WHERE w.business_id = $1
		  AND w.deleted_at IS NULL
		ORDER BY w.created_at DESC, w.name ASC
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list warranties: %w", err)
	}
	defer rows.Close()

	items := make([]models.Warranty, 0)
	for rows.Next() {
		var warranty models.Warranty
		if err := rows.Scan(
			&warranty.ID,
			&warranty.BusinessID,
			&warranty.Name,
			&warranty.Description,
			&warranty.DurationValue,
			&warranty.DurationUnit,
			&warranty.AddedByID,
			&warranty.AddedBy,
			&warranty.AddedAt,
			&warranty.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan warranty: %w", err)
		}
		items = append(items, warranty)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate warranties: %w", err)
	}

	log.Printf("list warranties: success business_id=%s count=%d", businessID, len(items))
	return items, nil
}

func CreateWarrantyRepository(pool *pgxpool.Pool, req CreateWarrantyInput) (*models.Warranty, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.DurationUnit = strings.TrimSpace(strings.ToLower(req.DurationUnit))
	req.AddedByID = strings.TrimSpace(req.AddedByID)
	req.AddedBy = strings.TrimSpace(req.AddedBy)

	if req.BusinessID == "" || req.Name == "" || req.DurationUnit == "" {
		return nil, ErrInvalidWarrantyInput
	}

	if req.DurationValue < 0 {
		return nil, ErrInvalidWarrantyInput
	}

	if req.DurationUnit != "days" && req.DurationUnit != "months" {
		return nil, ErrInvalidWarrantyInput
	}

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM product_warranties
			WHERE business_id = $1
			  AND LOWER(name) = LOWER($2)
			  AND deleted_at IS NULL
		)
	`, req.BusinessID, req.Name).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check warranty duplicate: %w", err)
	}
	if exists {
		return nil, ErrWarrantyAlreadyExists
	}

	var warranty models.Warranty
	err := pool.QueryRow(ctx, `
		INSERT INTO product_warranties (
			business_id,
			name,
			description,
			duration_value,
			duration_unit,
			added_by
		)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::uuid)
		RETURNING
			id::text,
			business_id::text,
			name,
			COALESCE(description, ''),
			duration_value,
			duration_unit,
			COALESCE(added_by::text, ''),
			created_at::text,
			updated_at::text
	`, req.BusinessID, req.Name, req.Description, req.DurationValue, req.DurationUnit, req.AddedByID).Scan(
		&warranty.ID,
		&warranty.BusinessID,
		&warranty.Name,
		&warranty.Description,
		&warranty.DurationValue,
		&warranty.DurationUnit,
		&warranty.AddedByID,
		&warranty.AddedAt,
		&warranty.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create warranty: %w", err)
	}

	warranty.AddedBy = req.AddedBy
	return &warranty, nil
}

func UpdateWarrantyRepository(pool *pgxpool.Pool, req UpdateWarrantyInput) (*models.Warranty, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.ID = strings.TrimSpace(req.ID)
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.DurationUnit = strings.TrimSpace(strings.ToLower(req.DurationUnit))

	if req.BusinessID == "" || req.ID == "" || req.Name == "" || req.DurationUnit == "" {
		return nil, ErrInvalidWarrantyInput
	}

	if req.DurationValue < 0 {
		return nil, ErrInvalidWarrantyInput
	}

	if req.DurationUnit != "days" && req.DurationUnit != "months" {
		return nil, ErrInvalidWarrantyInput
	}

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM product_warranties
			WHERE business_id = $1
			  AND id::text <> $2
			  AND LOWER(name) = LOWER($3)
			  AND deleted_at IS NULL
		)
	`, req.BusinessID, req.ID, req.Name).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check warranty duplicate: %w", err)
	}
	if exists {
		return nil, ErrWarrantyAlreadyExists
	}

	var warranty models.Warranty
	err := pool.QueryRow(ctx, `
		UPDATE product_warranties
		SET name = $3,
			description = $4,
			duration_value = $5,
			duration_unit = $6
		WHERE business_id = $1
		  AND id::text = $2
		  AND deleted_at IS NULL
		RETURNING
			id::text,
			business_id::text,
			name,
			COALESCE(description, ''),
			duration_value,
			duration_unit,
			COALESCE(added_by::text, ''),
			created_at::text,
			updated_at::text
	`, req.BusinessID, req.ID, req.Name, req.Description, req.DurationValue, req.DurationUnit).Scan(
		&warranty.ID,
		&warranty.BusinessID,
		&warranty.Name,
		&warranty.Description,
		&warranty.DurationValue,
		&warranty.DurationUnit,
		&warranty.AddedByID,
		&warranty.AddedAt,
		&warranty.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrWarrantyNotFound
		}
		return nil, fmt.Errorf("update warranty: %w", err)
	}

	return &warranty, nil
}

func DeleteWarrantyRepository(pool *pgxpool.Pool, businessID, warrantyID, deletedBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	warrantyID = strings.TrimSpace(warrantyID)
	deletedBy = strings.TrimSpace(deletedBy)
	if businessID == "" || warrantyID == "" {
		return ErrInvalidWarrantyInput
	}

	result, err := pool.Exec(ctx, `
		UPDATE product_warranties
		SET deleted = TRUE,
			deleted_at = NOW(),
			deleted_by = NULLIF($3, '')::uuid
		WHERE business_id = $1
		  AND id::text = $2
		  AND deleted_at IS NULL
	`, businessID, warrantyID, deletedBy)
	if err != nil {
		return fmt.Errorf("delete warranty: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrWarrantyNotFound
	}

	return nil
}
