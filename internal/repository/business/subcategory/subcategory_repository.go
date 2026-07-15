package subcategory

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

func ListSubCategoriesRepository(pool *pgxpool.Pool, businessID string) ([]models.SubCategory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	rows, err := pool.Query(ctx, `
		SELECT
			sc.uuid_id::text,
			sc.business_id::text,
			sc.parent_category_id::text,
			COALESCE(pc.name, '') AS parent_category_name,
			sc.sub_category_code,
			sc.name,
			COALESCE(sc.description, ''),
			COALESCE(sc.meta_title, ''),
			COALESCE(sc.meta_description, ''),
			COALESCE(sc.image_url, ''),
			sc.active,
			sc.featured,
			sc.sort_order,
			sc.created_at::text,
			sc.updated_at::text
		FROM product_sub_categories sc
		INNER JOIN product_categories pc ON pc.id = sc.parent_category_id AND pc.deleted_at IS NULL
		WHERE sc.business_id = $1
		  AND sc.deleted_at IS NULL
		ORDER BY sc.created_at DESC, sc.name ASC
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list sub categories: %w", err)
	}
	defer rows.Close()

	items := make([]models.SubCategory, 0)
	for rows.Next() {
		var item models.SubCategory
		if err := rows.Scan(
			&item.ID,
			&item.BusinessID,
			&item.ParentCategoryID,
			&item.ParentCategoryName,
			&item.SubCategoryCode,
			&item.Name,
			&item.Description,
			&item.MetaTitle,
			&item.MetaDescription,
			&item.ImageURL,
			&item.Active,
			&item.Featured,
			&item.SortOrder,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan sub category: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sub categories: %w", err)
	}

	log.Printf("list sub categories: success business_id=%s count=%d", businessID, len(items))
	return items, nil
}

func CreateSubCategoryRepository(pool *pgxpool.Pool, req CreateSubCategoryInput) (*models.SubCategory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.ParentCategoryID = strings.TrimSpace(req.ParentCategoryID)
	req.SubCategoryCode = strings.TrimSpace(req.SubCategoryCode)
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.MetaTitle = strings.TrimSpace(req.MetaTitle)
	req.MetaDescription = strings.TrimSpace(req.MetaDescription)
	req.ImageURL = strings.TrimSpace(req.ImageURL)

	if req.BusinessID == "" || req.ParentCategoryID == "" || req.Name == "" {
		return nil, ErrInvalidSubCategoryInput
	}

	if req.SubCategoryCode == "" {
		req.SubCategoryCode = generateSubCategoryCode(req.Name)
	}

	var parentExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM product_categories
			WHERE business_id = $1
			  AND uuid_id::text = $2
			  AND deleted_at IS NULL
		)
	`, req.BusinessID, req.ParentCategoryID).Scan(&parentExists); err != nil {
		return nil, fmt.Errorf("check parent category: %w", err)
	}
	if !parentExists {
		return nil, ErrInvalidSubCategoryInput
	}

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM product_sub_categories
			WHERE business_id = $1
			  AND parent_category_id::text = $2
			  AND LOWER(name) = LOWER($3)
			  AND deleted_at IS NULL
		)
	`, req.BusinessID, req.ParentCategoryID, req.Name).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check sub category duplicate name: %w", err)
	}
	if exists {
		return nil, ErrSubCategoryAlreadyExists
	}

	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM product_sub_categories
			WHERE business_id = $1
			  AND sub_category_code = $2
			  AND deleted_at IS NULL
		)
	`, req.BusinessID, req.SubCategoryCode).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check sub category duplicate code: %w", err)
	}
	if exists {
		return nil, ErrSubCategoryAlreadyExists
	}

	var item models.SubCategory
	err := pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO product_sub_categories (
				business_id,
				parent_category_id,
				sub_category_code,
				name,
				description,
				meta_title,
				meta_description,
				image_url,
				active,
				featured,
				sort_order
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), $9, $10, $11)
			RETURNING
				uuid_id::text AS id,
				business_id::text AS business_id,
				parent_category_id::text AS parent_category_id,
				sub_category_code AS sub_category_code,
				name AS name,
				COALESCE(description, '') AS description,
				COALESCE(meta_title, '') AS meta_title,
				COALESCE(meta_description, '') AS meta_description,
				COALESCE(image_url, '') AS image_url,
				active AS active,
				featured AS featured,
				sort_order AS sort_order,
				created_at::text AS created_at,
				updated_at::text AS updated_at
		)
		SELECT
			i.id,
			i.business_id,
			i.parent_category_id,
			COALESCE(pc.name, '') AS parent_category_name,
			i.sub_category_code,
			i.name,
			i.description,
			i.meta_title,
			i.meta_description,
			i.image_url,
			i.active,
			i.featured,
			i.sort_order,
			i.created_at,
			i.updated_at
		FROM inserted i
		LEFT JOIN product_categories pc ON pc.id::text = i.parent_category_id
	`, req.BusinessID, req.ParentCategoryID, req.SubCategoryCode, req.Name, nullIfBlank(req.Description), nullIfBlank(req.MetaTitle), nullIfBlank(req.MetaDescription), req.ImageURL, req.Active, req.Featured, req.SortOrder).Scan(
		&item.ID,
		&item.BusinessID,
		&item.ParentCategoryID,
		&item.ParentCategoryName,
		&item.SubCategoryCode,
		&item.Name,
		&item.Description,
		&item.MetaTitle,
		&item.MetaDescription,
		&item.ImageURL,
		&item.Active,
		&item.Featured,
		&item.SortOrder,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create sub category: %w", err)
	}

	return &item, nil
}

func UpdateSubCategoryRepository(pool *pgxpool.Pool, req UpdateSubCategoryInput) (*models.SubCategory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.ID = strings.TrimSpace(req.ID)
	req.ParentCategoryID = strings.TrimSpace(req.ParentCategoryID)
	req.SubCategoryCode = strings.TrimSpace(req.SubCategoryCode)
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.MetaTitle = strings.TrimSpace(req.MetaTitle)
	req.MetaDescription = strings.TrimSpace(req.MetaDescription)
	req.ImageURL = strings.TrimSpace(req.ImageURL)

	if req.BusinessID == "" || req.ID == "" || req.ParentCategoryID == "" || req.Name == "" {
		return nil, ErrInvalidSubCategoryInput
	}

	if req.SubCategoryCode == "" {
		req.SubCategoryCode = generateSubCategoryCode(req.Name)
	}

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM product_categories
			WHERE business_id = $1
			  AND uuid_id::text = $2
			  AND deleted_at IS NULL
		)
	`, req.BusinessID, req.ParentCategoryID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check parent category: %w", err)
	}
	if !exists {
		return nil, ErrInvalidSubCategoryInput
	}

	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM product_sub_categories
			WHERE business_id = $1
			  AND uuid_id::text <> $2
			  AND parent_category_id::text = $3
			  AND LOWER(name) = LOWER($4)
			  AND deleted_at IS NULL
		)
	`, req.BusinessID, req.ID, req.ParentCategoryID, req.Name).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check sub category duplicate name: %w", err)
	}
	if exists {
		return nil, ErrSubCategoryAlreadyExists
	}

	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM product_sub_categories
			WHERE business_id = $1
			  AND uuid_id::text <> $2
			  AND sub_category_code = $3
			  AND deleted_at IS NULL
		)
	`, req.BusinessID, req.ID, req.SubCategoryCode).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check sub category duplicate code: %w", err)
	}
	if exists {
		return nil, ErrSubCategoryAlreadyExists
	}

	var item models.SubCategory
	err := pool.QueryRow(ctx, `
		UPDATE product_sub_categories
		SET parent_category_id = $3,
			sub_category_code = $4,
			name = $5,
			description = $6,
			meta_title = $7,
			meta_description = $8,
			image_url = NULLIF($9, ''),
			active = $10,
			featured = $11,
			sort_order = $12
		WHERE business_id = $1
		  AND uuid_id::text = $2
		  AND deleted_at IS NULL
		RETURNING
			uuid_id::text,
			business_id::text,
			parent_category_id::text,
			sub_category_code,
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
	`, req.BusinessID, req.ID, req.ParentCategoryID, req.SubCategoryCode, req.Name, nullIfBlank(req.Description), nullIfBlank(req.MetaTitle), nullIfBlank(req.MetaDescription), req.ImageURL, req.Active, req.Featured, req.SortOrder).Scan(
		&item.ID,
		&item.BusinessID,
		&item.ParentCategoryID,
		&item.SubCategoryCode,
		&item.Name,
		&item.Description,
		&item.MetaTitle,
		&item.MetaDescription,
		&item.ImageURL,
		&item.Active,
		&item.Featured,
		&item.SortOrder,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSubCategoryNotFound
		}
		return nil, fmt.Errorf("update sub category: %w", err)
	}

	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(name, '')
		FROM product_categories
		WHERE uuid_id::text = $1
		  AND deleted_at IS NULL
	`, item.ParentCategoryID).Scan(&item.ParentCategoryName); err != nil {
		return nil, fmt.Errorf("load parent category: %w", err)
	}

	return &item, nil
}

func DeleteSubCategoryRepository(pool *pgxpool.Pool, businessID, subCategoryID, deletedBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	subCategoryID = strings.TrimSpace(subCategoryID)
	deletedBy = strings.TrimSpace(deletedBy)
	if businessID == "" || subCategoryID == "" {
		return ErrInvalidSubCategoryInput
	}

	result, err := pool.Exec(ctx, `
		UPDATE product_sub_categories
		SET deleted = TRUE,
			deleted_at = NOW(),
			deleted_by = NULLIF($3, '')::uuid
		WHERE business_id = $1
		  AND uuid_id::text = $2
		  AND deleted_at IS NULL
	`, businessID, subCategoryID, deletedBy)
	if err != nil {
		return fmt.Errorf("delete sub category: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSubCategoryNotFound
	}

	return nil
}

func generateSubCategoryCode(name string) string {
	code := strings.ToUpper(strings.TrimSpace(name))
	code = strings.ReplaceAll(code, " ", "-")
	code = strings.ReplaceAll(code, "/", "-")
	code = strings.ReplaceAll(code, "_", "-")
	code = strings.Trim(code, "-")
	if len(code) > 50 {
		code = code[:50]
	}
	return code
}

func nullIfBlank(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}
