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
			COALESCE(h.buying_price, 0),
			COALESCE(h.selling_price, 0),
			COALESCE(h.reason, ''),
			COALESCE(h.changed_by::text, ''),
			COALESCE(u.full_name, 'System'),
			h.created_at::text
		FROM product_price_history h
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
		if err := rows.Scan(&item.ID, &item.ProductID, &item.BuyingPrice, &item.SellingPrice, &reason, &item.ChangedByID, &item.ChangedByName, &item.CreatedAt); err != nil {
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
