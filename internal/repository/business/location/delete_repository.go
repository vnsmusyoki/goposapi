package location

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

func DeleteBusinessLocationRepository(pool *pgxpool.Pool, businessID, locationID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	locationID = strings.TrimSpace(locationID)
	if businessID == "" || locationID == "" {
		return ErrInvalidBusinessLocationInput
	}

	result, err := pool.Exec(ctx, `
		DELETE FROM business_locations
		WHERE business_id = $1
		  AND id::text = $2
	`, businessID, locationID)
	if err != nil {
		return fmt.Errorf("delete business location: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrBusinessLocationNotFound
	}

	return nil
}
