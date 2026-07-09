package admin

import (
	"context"
	"errors"
	"log"
	"pos/internal/models"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func CreatePackageRepository(
	pool *pgxpool.Pool,
	req CreatePackageRequest,
) (*models.Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("create package: start name=%q slug=%q currency=%q price=%.2f trial_days=%d max_users=%d max_branches=%d max_products=%d",
		req.Name, req.Slug, req.Currency, req.Price, req.TrialDays, req.MaxUsers, req.MaxBranches, req.MaxProducts)

	intervalID, err := billingIntervalIDByCode(ctx, pool, req.BillingIntervalCode)
	if err != nil {
		log.Printf("create package: billing interval lookup failed code=%q err=%v", req.BillingIntervalCode, err)
		return nil, err
	}
	log.Printf("create package: billing interval resolved code=%q id=%s", req.BillingIntervalCode, intervalID)

	var exists bool
	err = pool.QueryRow(
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
		log.Printf("create package: duplicate check failed name=%q slug=%q err=%v", req.Name, req.Slug, err)
		return nil, err
	}

	if exists {
		log.Printf("create package: duplicate detected name=%q slug=%q", req.Name, req.Slug)
		return nil, ErrPackageAlreadyExists
	}

	query := `
		INSERT INTO packages (
			name,
			slug,
			description,
			price,
			currency,
			billing_interval_id,
			trial_days,
			max_users,
			max_branches,
			max_products
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING
			id,
			name,
			slug,
			description,
			price,
			currency,
			billing_interval_id,
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
		intervalID,
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
		&pkg.BillingIntervalID,
		&pkg.TrialDays,
		&pkg.MaxUsers,
		&pkg.MaxBranches,
		&pkg.MaxProducts,
	)

	if err != nil {
		log.Printf("create package: insert failed name=%q slug=%q err=%v", req.Name, req.Slug, err)
		return nil, err
	}

	log.Printf("create package: success id=%s name=%q slug=%q", pkg.Id, pkg.Name, pkg.Slug)
	return &pkg, nil
}

func billingIntervalIDByCode(ctx context.Context, q queryer, code string) (string, error) {
	var intervalID string
	if err := q.QueryRow(ctx, `
		SELECT id::text
		FROM billing_intervals
		WHERE code = $1
		  AND is_active = TRUE
		LIMIT 1
	`, strings.ToLower(strings.TrimSpace(code))).Scan(&intervalID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrBillingIntervalNotFound
		}
		return "", err
	}

	return intervalID, nil
}
