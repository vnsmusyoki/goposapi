package brand

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

func ListBrandsRepository(pool *pgxpool.Pool, businessID string) ([]models.Brand, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	rows, err := pool.Query(ctx, `
		SELECT
			b.id::text,
			b.business_id::text,
			b.name,
			COALESCE(b.short_description, ''),
			COALESCE(b.added_by::text, ''),
			COALESCE(u.full_name, ''),
			b.created_at::text,
			b.updated_at::text
		FROM product_brands b
		LEFT JOIN users u ON u.id = b.added_by
		WHERE b.business_id = $1
		  AND b.deleted_at IS NULL
		ORDER BY b.created_at DESC, b.name ASC
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list brands: %w", err)
	}
	defer rows.Close()

	items := make([]models.Brand, 0)
	for rows.Next() {
		var brand models.Brand
		if err := rows.Scan(
			&brand.ID,
			&brand.BusinessID,
			&brand.Name,
			&brand.ShortDescription,
			&brand.AddedByID,
			&brand.AddedBy,
			&brand.AddedAt,
			&brand.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan brand: %w", err)
		}
		items = append(items, brand)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate brands: %w", err)
	}

	log.Printf("list brands: success business_id=%s count=%d", businessID, len(items))
	return items, nil
}

func CreateBrandRepository(pool *pgxpool.Pool, req CreateBrandInput) (*models.Brand, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.Name = strings.TrimSpace(req.Name)
	req.ShortDescription = strings.TrimSpace(req.ShortDescription)
	req.AddedByID = strings.TrimSpace(req.AddedByID)
	req.AddedBy = strings.TrimSpace(req.AddedBy)

	if req.BusinessID == "" || req.Name == "" {
		return nil, ErrInvalidBrandInput
	}

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM product_brands
			WHERE business_id = $1
			  AND LOWER(name) = LOWER($2)
			  AND deleted_at IS NULL
		)
	`, req.BusinessID, req.Name).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check brand duplicate: %w", err)
	}
	if exists {
		return nil, ErrBrandAlreadyExists
	}

	var brand models.Brand
	err := pool.QueryRow(ctx, `
		INSERT INTO product_brands (
			business_id,
			name,
			short_description,
			added_by
		)
		VALUES ($1, $2, $3, NULLIF($4, '')::uuid)
		RETURNING
			id::text,
			business_id::text,
			name,
			COALESCE(short_description, ''),
			COALESCE(added_by::text, ''),
			created_at::text,
			updated_at::text
	`, req.BusinessID, req.Name, req.ShortDescription, req.AddedByID).Scan(
		&brand.ID,
		&brand.BusinessID,
		&brand.Name,
		&brand.ShortDescription,
		&brand.AddedByID,
		&brand.AddedAt,
		&brand.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create brand: %w", err)
	}

	brand.AddedBy = req.AddedBy
	return &brand, nil
}

func UpdateBrandRepository(pool *pgxpool.Pool, req UpdateBrandInput) (*models.Brand, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.ID = strings.TrimSpace(req.ID)
	req.Name = strings.TrimSpace(req.Name)
	req.ShortDescription = strings.TrimSpace(req.ShortDescription)

	if req.BusinessID == "" || req.ID == "" || req.Name == "" {
		return nil, ErrInvalidBrandInput
	}

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM product_brands
			WHERE business_id = $1
			  AND id::text <> $2
			  AND LOWER(name) = LOWER($3)
			  AND deleted_at IS NULL
		)
	`, req.BusinessID, req.ID, req.Name).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check brand duplicate: %w", err)
	}
	if exists {
		return nil, ErrBrandAlreadyExists
	}

	var brand models.Brand
	err := pool.QueryRow(ctx, `
		UPDATE product_brands
		SET name = $3,
			short_description = $4
		WHERE business_id = $1
		  AND id::text = $2
		  AND deleted_at IS NULL
		RETURNING
			id::text,
			business_id::text,
			name,
			COALESCE(short_description, ''),
			COALESCE(added_by::text, ''),
			created_at::text,
			updated_at::text
	`, req.BusinessID, req.ID, req.Name, req.ShortDescription).Scan(
		&brand.ID,
		&brand.BusinessID,
		&brand.Name,
		&brand.ShortDescription,
		&brand.AddedByID,
		&brand.AddedAt,
		&brand.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrBrandNotFound
		}
		return nil, fmt.Errorf("update brand: %w", err)
	}

	return &brand, nil
}

func DeleteBrandRepository(pool *pgxpool.Pool, businessID, brandID, deletedBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	brandID = strings.TrimSpace(brandID)
	deletedBy = strings.TrimSpace(deletedBy)
	if businessID == "" || brandID == "" {
		return ErrInvalidBrandInput
	}

	result, err := pool.Exec(ctx, `
		UPDATE product_brands
		SET deleted = TRUE,
			deleted_at = NOW(),
			deleted_by = NULLIF($3, '')::uuid
		WHERE business_id = $1
		  AND id::text = $2
		  AND deleted_at IS NULL
	`, businessID, brandID, deletedBy)
	if err != nil {
		return fmt.Errorf("delete brand: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrBrandNotFound
	}

	return nil
}
