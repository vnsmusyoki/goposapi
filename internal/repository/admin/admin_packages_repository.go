package admin

import (
	"context"
	"pos/internal/models"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

func CreatePackageRepository(
	pool *pgxpool.Pool,
	req CreatePackageRequest,
) (*models.Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var exists bool
	err := pool.QueryRow(
		ctx,
		`
			SELECT EXISTS (
				SELECT 1
				FROM packages
				WHERE name = $1 OR slug = $2
			)
		`,
		req.Name,
		req.Slug,
	).Scan(&exists)

	if err != nil {
		return nil, err
	}
	
	if exists {
		return nil, ErrPackageAlreadyExists
	}

	query := `
		INSERT INTO packages (
			name,
			slug,
			description,
			price,
			currency,
			trial_days,
			max_users,
			max_branches,
			max_products
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING
			id,
			name,
			slug,
			description,
			price,
			currency,
			trial_days,
			max_users,
			max_branches,
			max_products
	`

	var pkg models.Package

	err = pool.QueryRow(
		ctx,
		query,
		req.Name,
		req.Slug,
		req.Description,
		req.Price,
		req.Currency,
		req.TrialDays,
		req.MaxUsers,
		req.MaxBranches,
		req.MaxProducts,
	).Scan(
		&pkg.Id,
		&pkg.Name,
		&pkg.Slug,
		&pkg.Description,
		&pkg.Price,
		&pkg.Currency,
		&pkg.TrialDays,
		&pkg.MaxUsers,
		&pkg.MaxBranches,
		&pkg.MaxProducts,
	)

	if err != nil {
		return nil, err
	}

	return &pkg, nil
}
