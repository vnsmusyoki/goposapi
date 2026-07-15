package product

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/models"
)

func ListProductsRepository(pool *pgxpool.Pool, businessID string, filters ListProductsFilters) ([]models.ProductListItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	filters.Search = strings.TrimSpace(filters.Search)
	filters.ProductType = strings.TrimSpace(strings.ToLower(filters.ProductType))
	filters.CategoryID = strings.TrimSpace(filters.CategoryID)
	filters.BrandID = strings.TrimSpace(filters.BrandID)
	filters.UnitID = strings.TrimSpace(filters.UnitID)
	filters.LocationID = strings.TrimSpace(filters.LocationID)
	filters.TaxType = strings.TrimSpace(strings.ToLower(filters.TaxType))
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	args := []any{businessID}
	where := `
		WHERE p.business_id = $1
		  AND p.deleted_at IS NULL
	`

	if filters.Search != "" {
		args = append(args, "%"+strings.ToLower(filters.Search)+"%")
		where += fmt.Sprintf(`
		  AND (
			LOWER(COALESCE(p.name, '')) LIKE $%d
			OR LOWER(COALESCE(p.sku, '')) LIKE $%d
			OR LOWER(COALESCE(p.barcode, '')) LIKE $%d
		  )
		`, len(args), len(args), len(args))
	}

	if filters.ProductType != "" && filters.ProductType != "all" {
		args = append(args, filters.ProductType)
		where += fmt.Sprintf(" AND p.product_type = $%d", len(args))
	}

	if filters.CategoryID != "" && filters.CategoryID != "all" {
		args = append(args, filters.CategoryID)
		where += fmt.Sprintf(" AND p.category_id = $%d::uuid", len(args))
	}

	if filters.BrandID != "" && filters.BrandID != "all" {
		args = append(args, filters.BrandID)
		where += fmt.Sprintf(" AND p.brand_id = $%d::uuid", len(args))
	}

	if filters.UnitID != "" && filters.UnitID != "all" {
		args = append(args, filters.UnitID)
		where += fmt.Sprintf(" AND p.unit_id = $%d::uuid", len(args))
	}

	if filters.LocationID != "" && filters.LocationID != "all" {
		args = append(args, filters.LocationID)
		where += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM product_locations pl2 WHERE pl2.product_id = p.id AND pl2.location_id = $%d::uuid AND pl2.deleted_at IS NULL)", len(args))
	}

	if filters.TaxType != "" && filters.TaxType != "all" {
		args = append(args, filters.TaxType)
		where += fmt.Sprintf(" AND p.tax_type = $%d", len(args))
	}

	if !filters.ShowNotForSelling {
		where += " AND p.is_for_selling = TRUE"
	}

	rows, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT
			p.id::text,
			p.name,
			p.sku,
			COALESCE(pi.image_url, ''),
			COALESCE(p.barcode, ''),
			p.product_type,
			COALESCE(p.unit_id::text, ''),
			COALESCE(u.name, ''),
			COALESCE(p.brand_id::text, ''),
			COALESCE(b.name, ''),
			COALESCE(p.category_id::text, ''),
			COALESCE(c.name, ''),
			COALESCE(p.sub_category_id::text, ''),
			COALESCE(sc.name, ''),
			COALESCE(array_agg(DISTINCT pl.location_id::text) FILTER (WHERE pl.location_id IS NOT NULL), '{}'::text[]),
			COALESCE(array_agg(DISTINCT bl.location_name) FILTER (WHERE bl.location_name IS NOT NULL), '{}'::text[]),
			p.manage_stock,
			COALESCE(p.alert_quantity, 0),
			p.is_for_selling,
			p.tax_type,
			COALESCE(p.tax_rate, 0),
			COALESCE(p.default_purchase_price, 0),
			COALESCE(p.default_selling_price, 0),
			COALESCE(p.profit_margin, 0),
			COALESCE(stock.current_stock, 0) AS current_stock,
			COALESCE(stock.current_stock_value, 0)::numeric AS current_stock_value,
			0::int AS total_units_sold,
			0::int AS total_units_transferred,
			0::int AS total_units_adjusted,
			p.created_at::text,
			p.updated_at::text,
			CASE WHEN p.deleted_at IS NULL THEN 'active' ELSE 'inactive' END
		FROM products p
		LEFT JOIN business_units u ON u.id = p.unit_id
		LEFT JOIN product_brands b ON b.id = p.brand_id
		LEFT JOIN product_categories c ON c.id = p.category_id
		LEFT JOIN product_sub_categories sc ON sc.uuid_id = p.sub_category_id
		LEFT JOIN product_locations pl ON pl.product_id = p.id AND pl.deleted_at IS NULL
		LEFT JOIN business_locations bl ON bl.id = pl.location_id
		LEFT JOIN LATERAL (
			SELECT
				COALESCE(ROUND(SUM(ib.quantity_available)), 0)::int AS current_stock,
				COALESCE(SUM(ib.quantity_available * COALESCE(p.default_purchase_price, 0)), 0)::numeric AS current_stock_value
			FROM inventory_balances ib
			WHERE ib.business_id = p.business_id
			  AND ib.product_id = p.id
			  %s
		) stock ON TRUE
		LEFT JOIN LATERAL (
			SELECT image_url
			FROM product_images
			WHERE product_id = p.id
			  AND deleted_at IS NULL
			ORDER BY is_primary DESC, sort_order ASC, created_at ASC
			LIMIT 1
		) pi ON TRUE
		%s
		GROUP BY
			p.id, u.name, b.name, c.name, sc.name, pi.image_url, stock.current_stock, stock.current_stock_value
		ORDER BY p.created_at DESC, p.name ASC
	`, func() string {
		if filters.LocationID != "" && filters.LocationID != "all" {
			return fmt.Sprintf("AND ib.location_id = $%d::uuid", len(args))
		}
		return ""
	}(), where), args...)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	items := make([]models.ProductListItem, 0)
	for rows.Next() {
		var item models.ProductListItem
		var sku sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&sku,
			&item.ImageURL,
			&item.Barcode,
			&item.ProductType,
			&item.UnitID,
			&item.UnitName,
			&item.BrandID,
			&item.BrandName,
			&item.CategoryID,
			&item.CategoryName,
			&item.SubCategoryID,
			&item.SubCategoryName,
			&item.LocationIDs,
			&item.LocationNames,
			&item.ManageStock,
			&item.AlertQuantity,
			&item.IsForSelling,
			&item.TaxType,
			&item.TaxRate,
			&item.DefaultPurchasePrice,
			&item.DefaultSellingPrice,
			&item.ProfitMargin,
			&item.CurrentStock,
			&item.CurrentStockValue,
			&item.TotalUnitsSold,
			&item.TotalUnitsTransferred,
			&item.TotalUnitsAdjusted,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.Status,
		); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		item.SKU = models.StringPtrFromNullString(sku)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}

	return items, nil
}
