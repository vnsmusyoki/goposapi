package expensecategory

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

func ListExpenseCategoriesRepository(pool *pgxpool.Pool, businessID string) ([]models.ExpenseCategory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	rows, err := pool.Query(ctx, `
		SELECT
			ec.id::text,
			ec.business_id::text,
			COALESCE(ec.parent_id::text, ''),
			COALESCE(parent.name, ''),
			ec.name,
			COALESCE(ec.code, ''),
			COALESCE(ec.description, ''),
			ec.is_active,
			ec.sort_order,
			COALESCE(ec.created_by::text, ''),
			COALESCE(u.full_name, ''),
			ec.created_at::text,
			ec.updated_at::text
		FROM expense_categories ec
		LEFT JOIN expense_categories parent ON parent.id = ec.parent_id
		LEFT JOIN users u ON u.id = ec.created_by
		WHERE ec.business_id = $1
		ORDER BY ec.sort_order ASC, ec.created_at DESC, ec.name ASC
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list expense categories: %w", err)
	}
	defer rows.Close()

	items := make([]models.ExpenseCategory, 0)
	for rows.Next() {
		var item models.ExpenseCategory
		if err := rows.Scan(
			&item.ID,
			&item.BusinessID,
			&item.ParentID,
			&item.ParentName,
			&item.Name,
			&item.Code,
			&item.Description,
			&item.Active,
			&item.SortOrder,
			&item.CreatedByID,
			&item.CreatedBy,
			&item.AddedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan expense category: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expense categories: %w", err)
	}

	log.Printf("list expense categories: success business_id=%s count=%d", businessID, len(items))
	return items, nil
}

func CreateExpenseCategoryRepository(pool *pgxpool.Pool, req CreateExpenseCategoryInput) (*models.ExpenseCategory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.ParentID = strings.TrimSpace(req.ParentID)
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.Description = strings.TrimSpace(req.Description)
	req.CreatedByID = strings.TrimSpace(req.CreatedByID)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)
	req.UpdatedByID = strings.TrimSpace(req.UpdatedByID)
	req.UpdatedBy = strings.TrimSpace(req.UpdatedBy)

	if req.BusinessID == "" || req.Name == "" {
		return nil, ErrInvalidExpenseCategoryInput
	}

	slug := generateExpenseCategorySlug(req.Name)
	if req.Code == "" {
		req.Code = generateExpenseCategoryCode(req.Name)
	}

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM expense_categories
			WHERE business_id = $1
			  AND LOWER(name) = LOWER($2)
		)
	`, req.BusinessID, req.Name).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check expense category duplicate name: %w", err)
	}
	if exists {
		return nil, ErrExpenseCategoryAlreadyExists
	}

	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM expense_categories
			WHERE business_id = $1
			  AND code = $2
		)
	`, req.BusinessID, req.Code).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check expense category duplicate code: %w", err)
	}
	if exists {
		return nil, ErrExpenseCategoryAlreadyExists
	}

	var item models.ExpenseCategory
	err := pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO expense_categories (
				business_id,
				parent_id,
				name,
				slug,
				code,
				description,
				is_active,
				sort_order,
				created_by,
				updated_by
			)
			VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, NULLIF($6, ''), $7, $8, NULLIF($9, '')::uuid, NULLIF($10, '')::uuid)
			RETURNING
				id::text AS id,
				business_id::text AS business_id,
				COALESCE(parent_id::text, '') AS parent_id,
				name AS name,
				COALESCE(code, '') AS code,
				COALESCE(description, '') AS description,
				is_active AS active,
				sort_order AS sort_order,
				COALESCE(created_by::text, '') AS created_by_id,
				COALESCE(created_by::text, '') AS created_by,
				created_at::text AS added_at,
				updated_at::text AS updated_at
		)
		SELECT
			i.id,
			i.business_id,
			i.parent_id,
			COALESCE(parent.name, ''),
			i.name,
			i.code,
			i.description,
			i.active,
			i.sort_order,
			i.created_by_id,
			i.created_by,
			i.added_at,
			i.updated_at
		FROM inserted i
		LEFT JOIN expense_categories parent ON parent.id::text = NULLIF(i.parent_id, '')
	`, req.BusinessID, req.ParentID, req.Name, slug, req.Code, req.Description, req.Active, req.SortOrder, req.CreatedByID, req.UpdatedByID).Scan(
		&item.ID,
		&item.BusinessID,
		&item.ParentID,
		&item.ParentName,
		&item.Name,
		&item.Code,
		&item.Description,
		&item.Active,
		&item.SortOrder,
		&item.CreatedByID,
		&item.CreatedBy,
		&item.AddedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create expense category: %w", err)
	}

	if req.CreatedBy != "" {
		item.CreatedBy = req.CreatedBy
	}

	return &item, nil
}

func UpdateExpenseCategoryRepository(pool *pgxpool.Pool, req UpdateExpenseCategoryInput) (*models.ExpenseCategory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.ID = strings.TrimSpace(req.ID)
	req.ParentID = strings.TrimSpace(req.ParentID)
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.Description = strings.TrimSpace(req.Description)
	req.UpdatedByID = strings.TrimSpace(req.UpdatedByID)
	req.UpdatedBy = strings.TrimSpace(req.UpdatedBy)

	if req.BusinessID == "" || req.ID == "" || req.Name == "" {
		return nil, ErrInvalidExpenseCategoryInput
	}

	if req.Code == "" {
		req.Code = generateExpenseCategoryCode(req.Name)
	}

	slug := generateExpenseCategorySlug(req.Name)

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM expense_categories
			WHERE business_id = $1
			  AND id::text <> $2
			  AND LOWER(name) = LOWER($3)
		)
	`, req.BusinessID, req.ID, req.Name).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check expense category duplicate name: %w", err)
	}
	if exists {
		return nil, ErrExpenseCategoryAlreadyExists
	}

	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM expense_categories
			WHERE business_id = $1
			  AND id::text <> $2
			  AND code = $3
		)
	`, req.BusinessID, req.ID, req.Code).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check expense category duplicate code: %w", err)
	}
	if exists {
		return nil, ErrExpenseCategoryAlreadyExists
	}

	var item models.ExpenseCategory
	err := pool.QueryRow(ctx, `
		WITH updated AS (
			UPDATE expense_categories
			SET parent_id = NULLIF($3, '')::uuid,
				name = $4,
				slug = $5,
				code = $6,
				description = NULLIF($7, ''),
				is_active = $8,
				sort_order = $9,
				updated_by = NULLIF($10, '')::uuid
			WHERE business_id = $1
			  AND id::text = $2
			RETURNING
				id::text AS id,
				business_id::text AS business_id,
				COALESCE(parent_id::text, '') AS parent_id,
				name AS name,
				COALESCE(code, '') AS code,
				COALESCE(description, '') AS description,
				is_active AS active,
				sort_order AS sort_order,
				COALESCE(created_by::text, '') AS created_by_id,
				COALESCE(created_by::text, '') AS created_by,
				created_at::text AS added_at,
				updated_at::text AS updated_at
		)
		SELECT
			u.id,
			u.business_id,
			u.parent_id,
			COALESCE(parent.name, ''),
			u.name,
			u.code,
			u.description,
			u.active,
			u.sort_order,
			u.created_by_id,
			u.created_by,
			u.added_at,
			u.updated_at
		FROM updated u
		LEFT JOIN expense_categories parent ON parent.id::text = NULLIF(u.parent_id, '')
	`, req.BusinessID, req.ID, req.ParentID, req.Name, slug, req.Code, req.Description, req.Active, req.SortOrder, req.UpdatedByID).Scan(
		&item.ID,
		&item.BusinessID,
		&item.ParentID,
		&item.ParentName,
		&item.Name,
		&item.Code,
		&item.Description,
		&item.Active,
		&item.SortOrder,
		&item.CreatedByID,
		&item.CreatedBy,
		&item.AddedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrExpenseCategoryNotFound
		}
		return nil, fmt.Errorf("update expense category: %w", err)
	}

	if req.UpdatedBy != "" {
		item.CreatedBy = req.UpdatedBy
	}

	return &item, nil
}

func DeleteExpenseCategoryRepository(pool *pgxpool.Pool, businessID, expenseCategoryID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	expenseCategoryID = strings.TrimSpace(expenseCategoryID)
	if businessID == "" || expenseCategoryID == "" {
		return ErrInvalidExpenseCategoryInput
	}

	result, err := pool.Exec(ctx, `
		DELETE FROM expense_categories
		WHERE business_id = $1
		  AND id::text = $2
	`, businessID, expenseCategoryID)
	if err != nil {
		return fmt.Errorf("delete expense category: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrExpenseCategoryNotFound
	}

	return nil
}

func generateExpenseCategorySlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "/", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.ReplaceAll(slug, "--", "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "expense-category"
	}
	if len(slug) > 150 {
		slug = slug[:150]
	}
	return slug
}

func generateExpenseCategoryCode(name string) string {
	code := strings.ToUpper(strings.TrimSpace(name))
	code = strings.ReplaceAll(code, " ", "-")
	code = strings.ReplaceAll(code, "/", "-")
	code = strings.ReplaceAll(code, "_", "-")
	code = strings.ReplaceAll(code, "--", "-")
	code = strings.Trim(code, "-")
	if code == "" {
		code = "EXPENSE"
	}
	code = "EXP-" + code
	if len(code) > 30 {
		code = code[:30]
	}
	return code
}
