package purchaseorder

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type PurchaseReturnableStockItem struct {
	ID                     string  `json:"id"`
	ProductID              string  `json:"productId"`
	ProductName            string  `json:"productName"`
	SKU                    string  `json:"sku"`
	SupplierID             string  `json:"supplierId"`
	SupplierName           string  `json:"supplierName"`
	LocationID             string  `json:"locationId"`
	LocationName           string  `json:"locationName"`
	LotNumber              string  `json:"lotNumber"`
	BatchNumber            string  `json:"batchNumber"`
	ExpiryDate             *string `json:"expiryDate"`
	SuppliedBySupplier     float64 `json:"suppliedBySupplier"`
	SoldAlreadyForSupplier float64 `json:"soldAlreadyForSupplier"`
	AvailableQuantity      float64 `json:"availableQuantity"`
	UnitPrice              float64 `json:"unitPrice"`
	UnitCostBeforeTax      float64 `json:"unitCostBeforeTax"`
	CurrentStock           float64 `json:"currentStock"`
	ReceivedAt             string  `json:"receivedAt"`
	SourceReference        string  `json:"sourceReference"`
	SourceID               string  `json:"sourceId"`
}

func SearchPurchaseReturnableStockRepository(pool *pgxpool.Pool, businessID, query, locationID, supplierID string) ([]PurchaseReturnableStockGroup, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	query = strings.TrimSpace(query)
	locationID = strings.TrimSpace(locationID)
	supplierID = strings.TrimSpace(supplierID)

	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}
	if query == "" {
		return []PurchaseReturnableStockGroup{}, nil
	}

	args := []any{businessID, "%" + strings.ToLower(query) + "%"}
	where := `
		WHERE ib.business_id = $1::uuid
		  AND ib.quantity_remaining > 0
		  AND ib.source_type = 'purchase_order'
		  AND (
			LOWER(COALESCE(p.name, '')) LIKE $2
			OR LOWER(COALESCE(p.sku, '')) LIKE $2
			OR LOWER(COALESCE(ib.lot_number, '')) LIKE $2
			OR LOWER(COALESCE(ib.batch_number, '')) LIKE $2
			OR LOWER(COALESCE(po.reference_number, '')) LIKE $2
		  )
	`

	if locationID != "" && locationID != "all" {
		args = append(args, locationID)
		where += fmt.Sprintf(" AND ib.location_id = $%d::uuid", len(args))
	}

	if supplierID != "" && supplierID != "all" {
		args = append(args, supplierID)
		where += fmt.Sprintf(" AND COALESCE(ib.supplier_id, po.supplier_id) = $%d::uuid", len(args))
	}

	rows, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT
			ib.id::text,
			p.id::text,
			p.name,
			COALESCE(p.sku, ''),
			COALESCE(ib.supplier_id::text, po.supplier_id::text, ''),
			COALESCE(
				NULLIF(bs.business_name, ''),
				NULLIF(TRIM(CONCAT_WS(' ', bs.prefix, bs.first_name, bs.middle_name, bs.last_name)), ''),
				COALESCE(bs.contact_id, '')
			),
			ib.location_id::text,
			COALESCE(loc.location_name, ''),
			COALESCE(ib.lot_number, ''),
			COALESCE(ib.batch_number, ''),
			COALESCE(ib.expiry_date::text, ''),
			COALESCE(ib.quantity_received, 0),
			COALESCE(GREATEST(ib.quantity_received - ib.quantity_remaining, 0), 0),
			COALESCE(ib.quantity_remaining, 0),
			COALESCE(ib.unit_cost, 0),
			COALESCE(ib.unit_cost, 0),
			COALESCE(balance.quantity_available, 0),
			COALESCE(ib.received_at::text, ''),
			COALESCE(po.reference_number, ''),
			COALESCE(ib.source_id::text, '')
		FROM inventory_batches ib
		JOIN products p ON p.id = ib.product_id
		LEFT JOIN purchase_orders po ON po.id = ib.source_id::uuid AND ib.source_type = 'purchase_order'
		LEFT JOIN business_suppliers bs ON bs.id = COALESCE(ib.supplier_id, po.supplier_id) AND bs.business_id = ib.business_id AND bs.deleted_at IS NULL
		LEFT JOIN business_locations loc ON loc.id = ib.location_id
		LEFT JOIN LATERAL (
			SELECT COALESCE(SUM(quantity_available), 0) AS quantity_available
			FROM inventory_balances
			WHERE business_id = ib.business_id
			  AND product_id = ib.product_id
			  AND location_id = ib.location_id
		) balance ON TRUE
		%s
		ORDER BY COALESCE(ib.expiry_date, DATE '9999-12-31') ASC, ib.received_at ASC, p.name ASC
	`, where), args...)
	if err != nil {
		return nil, fmt.Errorf("search purchase returnable stock: %w", err)
	}
	defer rows.Close()

	items := make([]PurchaseReturnableStockItem, 0)
	for rows.Next() {
		var item PurchaseReturnableStockItem
		var expiry sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.ProductID,
			&item.ProductName,
			&item.SKU,
			&item.SupplierID,
			&item.SupplierName,
			&item.LocationID,
			&item.LocationName,
			&item.LotNumber,
			&item.BatchNumber,
			&expiry,
			&item.SuppliedBySupplier,
			&item.SoldAlreadyForSupplier,
			&item.AvailableQuantity,
			&item.UnitPrice,
			&item.UnitCostBeforeTax,
			&item.CurrentStock,
			&item.ReceivedAt,
			&item.SourceReference,
			&item.SourceID,
		); err != nil {
			return nil, fmt.Errorf("scan purchase returnable stock: %w", err)
		}
		if expiry.Valid {
			value := expiry.String
			item.ExpiryDate = &value
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purchase returnable stock: %w", err)
	}

	return groupPurchaseReturnableStockItems(items), nil
}

func groupPurchaseReturnableStockItems(items []PurchaseReturnableStockItem) []PurchaseReturnableStockGroup {
	type groupedAccumulator struct {
		group PurchaseReturnableStockGroup
	}

	groups := make(map[string]*groupedAccumulator)
	order := make([]string, 0)

	for _, item := range items {
		groupKey := fmt.Sprintf("%s::%s", item.ProductID, item.SupplierID)
		accumulator, ok := groups[groupKey]
		if !ok {
			accumulator = &groupedAccumulator{
				group: PurchaseReturnableStockGroup{
					GroupKey:               groupKey,
					ProductID:              item.ProductID,
					ProductName:            item.ProductName,
					SKU:                    item.SKU,
					SupplierID:             item.SupplierID,
					SupplierName:           item.SupplierName,
					LocationName:           item.LocationName,
					SuppliedBySupplier:     0,
					SoldAlreadyForSupplier: 0,
					AvailableQuantity:      0,
					UnitPrice:              item.UnitPrice,
				},
			}
			groups[groupKey] = accumulator
			order = append(order, groupKey)
		}

		accumulator.group.SuppliedBySupplier += item.SuppliedBySupplier
		accumulator.group.SoldAlreadyForSupplier += item.SoldAlreadyForSupplier
		accumulator.group.AvailableQuantity += item.AvailableQuantity
		accumulator.group.UnitPrice = item.UnitPrice
	}

	grouped := make([]PurchaseReturnableStockGroup, 0, len(order))
	for _, key := range order {
		grouped = append(grouped, groups[key].group)
	}

	sort.SliceStable(grouped, func(i, j int) bool {
		if grouped[i].ProductName == grouped[j].ProductName {
			return grouped[i].SupplierName < grouped[j].SupplierName
		}
		return grouped[i].ProductName < grouped[j].ProductName
	})

	return grouped
}
