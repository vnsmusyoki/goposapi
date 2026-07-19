package product

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

func ListProductPriceHistoryRepository(pool *pgxpool.Pool, businessID, productID string) ([]ProductPriceHistoryItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	productID = strings.TrimSpace(productID)
	if businessID == "" || productID == "" {
		return nil, ErrBusinessNotResolved
	}

	rows, err := pool.Query(ctx, `
		SELECT
			h.id::text,
			h.product_id::text,
			h.product_price_id::text,
			h.action,
			h.price_type,
			h.min_quantity,
			h.old_price,
			h.new_price,
			COALESCE(h.location_id::text, ''),
			COALESCE(h.customer_group, ''),
			COALESCE(h.starts_at::text, ''),
			COALESCE(h.ends_at::text, ''),
			h.active,
			h.priority,
			COALESCE(h.reason, ''),
			COALESCE(h.changed_by::text, ''),
			COALESCE(u.full_name, 'System'),
			h.created_at::text
		FROM product_price_rule_history h
		LEFT JOIN users u ON u.id = h.changed_by
		WHERE h.business_id = $1
		  AND h.product_id::text = $2
		ORDER BY h.created_at DESC
	`, businessID, productID)
	if err != nil {
		return nil, fmt.Errorf("list product price history: %w", err)
	}
	defer rows.Close()

	items := make([]ProductPriceHistoryItem, 0)
	for rows.Next() {
		var item ProductPriceHistoryItem
		var reason sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.ProductID,
			&item.ProductPriceID,
			&item.Action,
			&item.PriceType,
			&item.MinQuantity,
			&item.OldPrice,
			&item.NewPrice,
			&item.LocationID,
			&item.CustomerGroup,
			&item.StartsAt,
			&item.EndsAt,
			&item.Active,
			&item.Priority,
			&reason,
			&item.ChangedByID,
			&item.ChangedByName,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product price history: %w", err)
		}
		item.Reason = reason
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product price history: %w", err)
	}

	return items, nil
}
