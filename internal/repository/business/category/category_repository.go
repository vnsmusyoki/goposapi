package category

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/models"
)

func CreateCategoryRepository(
	pool *pgxpool.Pool,
	req CreateCategoryInput,
) (*models.Category, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.Name = strings.TrimSpace(req.Name)
	req.CategoryCode = strings.TrimSpace(req.CategoryCode)
	req.Description = strings.TrimSpace(req.Description)
	req.MetaTitle = strings.TrimSpace(req.MetaTitle)
	req.MetaDescription = strings.TrimSpace(req.MetaDescription)
	var err error
	req.ImageURL, err = normalizeCategoryImageDataURL(req.ImageURL)
	if err != nil {
		return nil, err
	}

	if req.BusinessID == "" || req.Name == "" {
		return nil, ErrInvalidCategoryInput
	}

	if req.CategoryCode == "" {
		req.CategoryCode = generateCategoryCode(req.Name)
	}

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM product_categories
			WHERE business_id = $1
			  AND LOWER(name) = LOWER($2)
			  AND deleted_at IS NULL
		)
	`, req.BusinessID, req.Name).Scan(&exists); err != nil {
		log.Printf("create category: duplicate name check failed business_id=%s name=%q err=%v", req.BusinessID, req.Name, err)
		return nil, err
	}
	if exists {
		return nil, ErrCategoryAlreadyExists
	}

	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM product_categories 
			WHERE business_id = $1 AND category_code = $2
			  AND deleted_at IS NULL
		)
	`, req.BusinessID, req.CategoryCode).Scan(&exists); err != nil {
		log.Printf("create category: duplicate code check failed business_id=%s code=%q err=%v", req.BusinessID, req.CategoryCode, err)
		return nil, err
	}
	if exists {
		return nil, ErrCategoryAlreadyExists
	}

	var category models.Category
	err = pool.QueryRow(
		ctx,
		`
		INSERT INTO product_categories (
			business_id,
			category_code,
			name,
			description,
			meta_title,
			meta_description,
			image_url,
			active,
			featured,
			sort_order
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING
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
	`,
		req.BusinessID,
		req.CategoryCode,
		req.Name,
		nullIfBlank(req.Description),
		nullIfBlank(req.MetaTitle),
		nullIfBlank(req.MetaDescription),
		nullIfBlank(req.ImageURL),
		req.Active,
		req.Featured,
		req.SortOrder,
	).Scan(
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
	)
	if err != nil {
		log.Printf("create category: insert failed business_id=%s name=%q code=%q err=%v", req.BusinessID, req.Name, req.CategoryCode, err)
		return nil, err
	}

	log.Printf("create category: success id=%s business_id=%s code=%q name=%q", category.ID, category.BusinessID, category.CategoryCode, category.Name)
	return &category, nil
}

func UpdateCategoryRepository(
	pool *pgxpool.Pool,
	req UpdateCategoryInput,
) (*models.Category, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.ID = strings.TrimSpace(req.ID)
	req.CategoryCode = strings.TrimSpace(req.CategoryCode)
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.MetaTitle = strings.TrimSpace(req.MetaTitle)
	req.MetaDescription = strings.TrimSpace(req.MetaDescription)
	req.ImageURL = strings.TrimSpace(req.ImageURL)

	if req.BusinessID == "" || req.ID == "" || req.Name == "" {
		return nil, ErrInvalidCategoryInput
	}

	if req.CategoryCode == "" {
		req.CategoryCode = generateCategoryCode(req.Name)
	}

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM product_categories
			WHERE business_id = $1
			  AND id::text <> $2
			  AND LOWER(name) = LOWER($3)
			  AND deleted_at IS NULL
		)
	`, req.BusinessID, req.ID, req.Name).Scan(&exists); err != nil {
		log.Printf("update category: duplicate name check failed business_id=%s id=%s name=%q err=%v", req.BusinessID, req.ID, req.Name, err)
		return nil, err
	}
	if exists {
		return nil, ErrCategoryAlreadyExists
	}

	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM product_categories
			WHERE business_id = $1
			  AND id::text <> $2
			  AND category_code = $3
			  AND deleted_at IS NULL
		)
	`, req.BusinessID, req.ID, req.CategoryCode).Scan(&exists); err != nil {
		log.Printf("update category: duplicate code check failed business_id=%s id=%s code=%q err=%v", req.BusinessID, req.ID, req.CategoryCode, err)
		return nil, err
	}
	if exists {
		return nil, ErrCategoryAlreadyExists
	}

	var category models.Category
	err := pool.QueryRow(
		ctx,
		`
			UPDATE product_categories
			SET category_code = $3,
				name = $4,
				description = $5,
				meta_title = $6,
				meta_description = $7,
				image_url = $8,
				active = $9,
				featured = $10,
				sort_order = $11
			WHERE business_id = $1
			  AND id::text = $2
			  AND deleted_at IS NULL
			RETURNING
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
		`,
		req.BusinessID,
		req.ID,
		req.CategoryCode,
		req.Name,
		nullIfBlank(req.Description),
		nullIfBlank(req.MetaTitle),
		nullIfBlank(req.MetaDescription),
		nullIfBlank(req.ImageURL),
		req.Active,
		req.Featured,
		req.SortOrder,
	).Scan(
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
	)
	if err != nil {
		log.Printf("update category: update failed business_id=%s id=%s name=%q code=%q err=%v", req.BusinessID, req.ID, req.Name, req.CategoryCode, err)
		return nil, err
	}

	log.Printf("update category: success id=%s business_id=%s code=%q name=%q", category.ID, category.BusinessID, category.CategoryCode, category.Name)
	return &category, nil
}

func generateCategoryCode(name string) string {
	code := strings.ToUpper(strings.TrimSpace(name))
	code = strings.ReplaceAll(code, " ", "-")
	code = strings.ReplaceAll(code, "/", "-")
	code = strings.ReplaceAll(code, "_", "-")
	code = strings.ReplaceAll(code, "--", "-")
	code = strings.Trim(code, "-")
	if code == "" {
		code = "CATEGORY"
	}
	code = "CAT-" + code
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
