package supplier

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type SupplierStockReportItem struct {
	ID                     string
	ProductID              string
	ProductName            string
	SKU                    string
	CategoryName           string
	LocationID             string
	LocationName           string
	SuppliedBySupplier     float64
	SoldAlreadyForSupplier float64
	QuantityAvailable      float64
	CostPrice              float64
	SellingPrice           float64
	Status                 string
	LastUpdated            string
}

func GetBusinessSupplierStockReportRepository(pool *pgxpool.Pool, businessID, supplierID string) ([]SupplierStockReportItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	supplierID = strings.TrimSpace(supplierID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}
	if supplierID == "" {
		return nil, fmt.Errorf("supplier id is required")
	}

	rows, err := pool.Query(ctx, `
		WITH supplier_batches AS (
			SELECT
				ib.id::text AS batch_id,
				ib.product_id::text AS product_id,
				ib.location_id::text AS location_id,
				COALESCE(ib.quantity_received, 0)::numeric AS quantity_received,
				COALESCE(ib.quantity_remaining, 0)::numeric AS quantity_remaining,
				COALESCE(ib.unit_cost, 0)::numeric AS unit_cost,
				COALESCE(ib.updated_at, ib.created_at)::timestamptz AS touched_at,
				COALESCE(p.name, '') AS product_name,
				COALESCE(p.sku, '') AS sku,
				COALESCE(p.default_selling_price, 0)::numeric AS selling_price,
				COALESCE(c.name, 'Uncategorized') AS category_name,
				COALESCE(bl.location_name, 'Unassigned') AS location_name,
				COALESCE(alloc.sold_quantity, 0)::numeric AS sold_quantity
			FROM inventory_batches ib
			LEFT JOIN purchase_orders po
				ON po.id = ib.source_id
			   AND ib.source_type = 'purchase_order'
			   AND po.deleted_at IS NULL
			LEFT JOIN products p
				ON p.id = ib.product_id
			   AND p.deleted_at IS NULL
			LEFT JOIN product_categories c
				ON c.id = p.category_id
			LEFT JOIN business_locations bl
				ON bl.id = ib.location_id
			LEFT JOIN (
				SELECT
					siba.inventory_batch_id,
					SUM(COALESCE(siba.allocated_quantity, 0)) AS sold_quantity
				FROM sale_item_batch_allocations siba
				INNER JOIN sales s
					ON s.id = siba.sale_id
				   AND s.business_id = siba.business_id
				   AND COALESCE(s.deleted, FALSE) = FALSE
				WHERE siba.business_id = $1
				GROUP BY siba.inventory_batch_id
			) alloc ON alloc.inventory_batch_id = ib.id
			WHERE ib.business_id = $1
			  AND COALESCE(ib.supplier_id, po.supplier_id)::text = $2
		)
		SELECT
			COALESCE(MIN(batch_id), product_id || ':' || location_id) AS id,
			product_id,
			product_name,
			sku,
			category_name,
			COALESCE(location_id, '') AS location_id,
			COALESCE(location_name, 'Unassigned') AS location_name,
			COALESCE(SUM(quantity_received), 0)::numeric AS supplied_by_supplier,
			COALESCE(SUM(sold_quantity), 0)::numeric AS sold_already_for_supplier,
			COALESCE(SUM(quantity_remaining), 0)::numeric AS quantity_available,
			COALESCE(AVG(unit_cost), 0)::numeric AS cost_price,
			COALESCE(MAX(selling_price), 0)::numeric AS selling_price,
			CASE
				WHEN COALESCE(SUM(quantity_remaining), 0) = 0 THEN 'out-of-stock'
				WHEN COALESCE(SUM(quantity_remaining), 0) <= GREATEST(1, ROUND(COALESCE(SUM(quantity_received), 0) * 0.1)) THEN 'critical'
				WHEN COALESCE(SUM(quantity_remaining), 0) <= GREATEST(1, ROUND(COALESCE(SUM(quantity_received), 0) * 0.25)) THEN 'low'
				ELSE 'healthy'
			END AS status,
			MAX(touched_at)::text AS last_updated
		FROM supplier_batches
		GROUP BY product_id, product_name, sku, category_name, location_id, location_name
		ORDER BY product_name ASC, location_name ASC
	`, businessID, supplierID)
	if err != nil {
		return nil, fmt.Errorf("load supplier stock report: %w", err)
	}
	defer rows.Close()

	items := make([]SupplierStockReportItem, 0)
	for rows.Next() {
		var item SupplierStockReportItem
		var touchedAt sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.ProductID,
			&item.ProductName,
			&item.SKU,
			&item.CategoryName,
			&item.LocationID,
			&item.LocationName,
			&item.SuppliedBySupplier,
			&item.SoldAlreadyForSupplier,
			&item.QuantityAvailable,
			&item.CostPrice,
			&item.SellingPrice,
			&item.Status,
			&touchedAt,
		); err != nil {
			return nil, fmt.Errorf("scan supplier stock report: %w", err)
		}
		if touchedAt.Valid {
			item.LastUpdated = touchedAt.String
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier stock report: %w", err)
	}

	return items, nil
}
