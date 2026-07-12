package category

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

func DeleteCategoryRepository(pool *pgxpool.Pool, businessID, categoryID, deletedBy string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	categoryID = strings.TrimSpace(categoryID)
	deletedBy = strings.TrimSpace(deletedBy)
	if businessID == "" || categoryID == "" {
		return 0, ErrInvalidCategoryInput
	}

	var subCategoryCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM product_sub_categories
		WHERE business_id = $1
		  AND parent_category_id::text = $2
		  AND deleted_at IS NULL
	`, businessID, categoryID).Scan(&subCategoryCount); err != nil {
		return 0, fmt.Errorf("check category subcategories: %w", err)
	}

	if subCategoryCount > 0 {
		return subCategoryCount, ErrCategoryHasSubCategories
	}

	result, err := pool.Exec(ctx, `
		UPDATE product_categories
		SET deleted = TRUE,
			deleted_at = NOW(),
			deleted_by = NULLIF($3, '')::uuid
		WHERE business_id = $1
		  AND id::text = $2
		  AND deleted_at IS NULL
	`, businessID, categoryID, deletedBy)
	if err != nil {
		return 0, fmt.Errorf("delete category: %w", err)
	}

	if result.RowsAffected() == 0 {
		return 0, ErrCategoryNotFound
	}

	return 0, nil
}
