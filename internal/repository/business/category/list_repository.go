package category

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/models"
)

func ListCategoriesRepository(
	pool *pgxpool.Pool,
	businessID string,
) ([]models.Category, error) {
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
			category_code,
			name,
			COALESCE(description, ''),
			COALESCE(meta_title, ''),
			COALESCE(meta_description, ''),
			COALESCE(image_url, ''),
			active,
			featured,
			sort_order,
			created_at::text,
			updated_at::text
		FROM product_categories
		WHERE business_id = $1
		  AND deleted_at IS NULL
		ORDER BY created_at DESC, name ASC
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	categories := make([]models.Category, 0)
	for rows.Next() {
		var category models.Category
		if err := rows.Scan(
			&category.ID,
			&category.BusinessID,
			&category.CategoryCode,
			&category.Name,
			&category.Description,
			&category.MetaTitle,
			&category.MetaDescription,
			&category.ImageURL,
			&category.Active,
			&category.Featured,
			&category.SortOrder,
			&category.CreatedAt,
			&category.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}

	log.Printf("list categories: success business_id=%s count=%d", businessID, len(categories))
	return categories, nil
}
