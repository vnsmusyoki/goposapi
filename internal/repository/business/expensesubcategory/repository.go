package expensesubcategory

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

func ListExpenseSubCategoriesRepository(pool *pgxpool.Pool, businessID string) ([]models.ExpenseSubCategory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	rows, err := pool.Query(ctx, `
		SELECT
			sc.id::text,
			sc.business_id::text,
			sc.expense_category_id::text,
			COALESCE(ec.name, ''),
			sc.name,
			COALESCE(sc.code, ''),
			COALESCE(sc.description, ''),
			sc.is_active,
			sc.sort_order,
			COALESCE(sc.created_by::text, ''),
			COALESCE(u.full_name, ''),
			sc.created_at::text,
			sc.updated_at::text
		FROM expense_sub_categories sc
		LEFT JOIN expense_categories ec ON ec.id = sc.expense_category_id
		LEFT JOIN users u ON u.id = sc.created_by
		WHERE sc.business_id = $1
		ORDER BY sc.sort_order ASC, sc.created_at DESC, sc.name ASC
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list expense sub categories: %w", err)
	}
	defer rows.Close()

	items := make([]models.ExpenseSubCategory, 0)
	for rows.Next() {
		var item models.ExpenseSubCategory
		if err := rows.Scan(
			&item.ID,
			&item.BusinessID,
			&item.ExpenseCategoryID,
			&item.ExpenseCategoryName,
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
			return nil, fmt.Errorf("scan expense sub category: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expense sub categories: %w", err)
	}

	log.Printf("list expense sub categories: success business_id=%s count=%d", businessID, len(items))
	return items, nil
}

func CreateExpenseSubCategoryRepository(pool *pgxpool.Pool, req CreateExpenseSubCategoryInput) (*models.ExpenseSubCategory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.ExpenseCategoryID = strings.TrimSpace(req.ExpenseCategoryID)
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.Description = strings.TrimSpace(req.Description)
	req.CreatedByID = strings.TrimSpace(req.CreatedByID)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)
	req.UpdatedByID = strings.TrimSpace(req.UpdatedByID)
	req.UpdatedBy = strings.TrimSpace(req.UpdatedBy)

	if req.BusinessID == "" || req.ExpenseCategoryID == "" || req.Name == "" {
		return nil, ErrInvalidExpenseSubCategoryInput
	}

	if req.Code == "" {
		req.Code = generateExpenseSubCategoryCode(req.Name)
	}

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM expense_categories
			WHERE business_id = $1
			  AND id::text = $2
		)
	`, req.BusinessID, req.ExpenseCategoryID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check expense category: %w", err)
	}
	if !exists {
		return nil, ErrInvalidExpenseSubCategoryInput
	}

	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM expense_sub_categories
			WHERE business_id = $1
			  AND expense_category_id::text = $2
			  AND LOWER(name) = LOWER($3)
		)
	`, req.BusinessID, req.ExpenseCategoryID, req.Name).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check expense sub category duplicate name: %w", err)
	}
	if exists {
		return nil, ErrExpenseSubCategoryAlreadyExists
	}

	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM expense_sub_categories
			WHERE business_id = $1
			  AND code = $2
		)
	`, req.BusinessID, req.Code).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check expense sub category duplicate code: %w", err)
	}
	if exists {
		return nil, ErrExpenseSubCategoryAlreadyExists
	}

	var item models.ExpenseSubCategory
	err := pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO expense_sub_categories (
				business_id,
				expense_category_id,
				name,
				code,
				description,
				is_active,
				sort_order,
				created_by,
				updated_by
			)
			VALUES ($1, $2::uuid, $3, $4, NULLIF($5, ''), $6, $7, NULLIF($8, '')::uuid, NULLIF($9, '')::uuid)
			RETURNING
				id::text AS id,
				business_id::text AS business_id,
				expense_category_id::text AS expense_category_id,
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
			i.expense_category_id,
			COALESCE(ec.name, ''),
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
		LEFT JOIN expense_categories ec ON ec.id::text = i.expense_category_id
	`, req.BusinessID, req.ExpenseCategoryID, req.Name, req.Code, req.Description, req.Active, req.SortOrder, req.CreatedByID, req.UpdatedByID).Scan(
		&item.ID,
		&item.BusinessID,
		&item.ExpenseCategoryID,
		&item.ExpenseCategoryName,
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
		return nil, fmt.Errorf("create expense sub category: %w", err)
	}

	if req.CreatedBy != "" {
		item.CreatedBy = req.CreatedBy
	}

	return &item, nil
}

func UpdateExpenseSubCategoryRepository(pool *pgxpool.Pool, req UpdateExpenseSubCategoryInput) (*models.ExpenseSubCategory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.ID = strings.TrimSpace(req.ID)
	req.ExpenseCategoryID = strings.TrimSpace(req.ExpenseCategoryID)
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.Description = strings.TrimSpace(req.Description)
	req.UpdatedByID = strings.TrimSpace(req.UpdatedByID)
	req.UpdatedBy = strings.TrimSpace(req.UpdatedBy)

	if req.BusinessID == "" || req.ID == "" || req.ExpenseCategoryID == "" || req.Name == "" {
		return nil, ErrInvalidExpenseSubCategoryInput
	}

	if req.Code == "" {
		req.Code = generateExpenseSubCategoryCode(req.Name)
	}

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM expense_categories
			WHERE business_id = $1
			  AND id::text = $2
		)
	`, req.BusinessID, req.ExpenseCategoryID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check expense category: %w", err)
	}
	if !exists {
		return nil, ErrInvalidExpenseSubCategoryInput
	}

	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM expense_sub_categories
			WHERE business_id = $1
			  AND id::text <> $2
			  AND expense_category_id::text = $3
			  AND LOWER(name) = LOWER($4)
		)
	`, req.BusinessID, req.ID, req.ExpenseCategoryID, req.Name).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check expense sub category duplicate name: %w", err)
	}
	if exists {
		return nil, ErrExpenseSubCategoryAlreadyExists
	}

	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM expense_sub_categories
			WHERE business_id = $1
			  AND id::text <> $2
			  AND code = $3
		)
	`, req.BusinessID, req.ID, req.Code).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check expense sub category duplicate code: %w", err)
	}
	if exists {
		return nil, ErrExpenseSubCategoryAlreadyExists
	}

	var item models.ExpenseSubCategory
	err := pool.QueryRow(ctx, `
		WITH updated AS (
			UPDATE expense_sub_categories
			SET expense_category_id = $3::uuid,
				name = $4,
				code = $5,
				description = NULLIF($6, ''),
				is_active = $7,
				sort_order = $8,
				updated_by = NULLIF($9, '')::uuid
			WHERE business_id = $1
			  AND id::text = $2
			RETURNING
				id::text AS id,
				business_id::text AS business_id,
				expense_category_id::text AS expense_category_id,
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
			u.expense_category_id,
			COALESCE(ec.name, ''),
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
		LEFT JOIN expense_categories ec ON ec.id::text = u.expense_category_id
	`, req.BusinessID, req.ID, req.ExpenseCategoryID, req.Name, req.Code, req.Description, req.Active, req.SortOrder, req.UpdatedByID).Scan(
		&item.ID,
		&item.BusinessID,
		&item.ExpenseCategoryID,
		&item.ExpenseCategoryName,
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
			return nil, ErrExpenseSubCategoryNotFound
		}
		return nil, fmt.Errorf("update expense sub category: %w", err)
	}

	if req.UpdatedBy != "" {
		item.CreatedBy = req.UpdatedBy
	}

	return &item, nil
}

func DeleteExpenseSubCategoryRepository(pool *pgxpool.Pool, businessID, expenseSubCategoryID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	expenseSubCategoryID = strings.TrimSpace(expenseSubCategoryID)
	if businessID == "" || expenseSubCategoryID == "" {
		return ErrInvalidExpenseSubCategoryInput
	}

	result, err := pool.Exec(ctx, `
		DELETE FROM expense_sub_categories
		WHERE business_id = $1
		  AND id::text = $2
	`, businessID, expenseSubCategoryID)
	if err != nil {
		return fmt.Errorf("delete expense sub category: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrExpenseSubCategoryNotFound
	}

	return nil
}

func generateExpenseSubCategoryCode(name string) string {
	code := strings.ToUpper(strings.TrimSpace(name))
	code = strings.ReplaceAll(code, " ", "-")
	code = strings.ReplaceAll(code, "/", "-")
	code = strings.ReplaceAll(code, "_", "-")
	code = strings.ReplaceAll(code, "--", "-")
	code = strings.Trim(code, "-")
	if code == "" {
		code = "SUB"
	}
	code = "ESC-" + code
	if len(code) > 50 {
		code = code[:50]
	}
	return code
}
