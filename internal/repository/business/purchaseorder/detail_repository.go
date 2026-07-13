package purchaseorder

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func GetPurchaseOrderDetailRepository(pool *pgxpool.Pool, businessID, purchaseOrderID string) (*PurchaseOrderDetail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	purchaseOrderID = strings.TrimSpace(purchaseOrderID)
	if businessID == "" || purchaseOrderID == "" {
		return nil, ErrBusinessNotResolved
	}

	order, err := getPurchaseOrderByID(ctx, pool, businessID, purchaseOrderID)
	if err != nil {
		return nil, err
	}

	supplier, err := getPurchaseOrderSupplierDetails(ctx, pool, order.SupplierID, businessID)
	if err != nil {
		return nil, err
	}

	businessDetails, err := getPurchaseOrderBusinessDetails(ctx, pool, businessID)
	if err != nil {
		return nil, err
	}

	locationDetails, err := getPurchaseOrderLocationDetails(ctx, pool, businessID, order.LocationID)
	if err != nil {
		return nil, err
	}

	items, err := listPurchaseOrderItemsByOrderID(ctx, pool, businessID, purchaseOrderID)
	if err != nil {
		return nil, err
	}

	additionalExpenses, err := listPurchaseOrderAdditionalExpensesByOrderID(ctx, pool, businessID, purchaseOrderID)
	if err != nil {
		return nil, err
	}

	activities, err := listPurchaseOrderLogsByOrderID(ctx, pool, businessID, purchaseOrderID)
	if err != nil {
		return nil, err
	}

	approvalActivities, err := listPurchaseOrderApprovalActivitiesByOrderID(ctx, pool, businessID, purchaseOrderID)
	if err != nil {
		return nil, err
	}

	activities = append(activities, approvalActivities...)
	sortPurchaseOrderActivities(activities)

	detail := &PurchaseOrderDetail{
		PurchaseOrder: *order,
		Supplier:      supplier,
		Business:      businessDetails,
		Location:      locationDetails,
		Items:         items,
		Activities:    activities,
	}
	detail.AdditionalExpenses = additionalExpenses
	return detail, nil
}

func getPurchaseOrderSupplierDetails(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, supplierID, businessID string) (PurchaseOrderSupplierDetails, error) {
	var (
		name    sql.NullString
		email   sql.NullString
		mobile  sql.NullString
		alt     sql.NullString
		land    sql.NullString
		line1   sql.NullString
		line2   sql.NullString
		city    sql.NullString
		state   sql.NullString
		country sql.NullString
		zipCode sql.NullString
	)

	err := querier.QueryRow(ctx, `
		SELECT
			COALESCE(
				NULLIF(bs.business_name, ''),
				NULLIF(TRIM(CONCAT_WS(' ', bs.prefix, bs.first_name, bs.middle_name, bs.last_name)), ''),
				bs.contact_id
			) AS name,
			COALESCE(bs.email, ''),
			COALESCE(bs.mobile, ''),
			COALESCE(bs.alternate_contact_number, ''),
			COALESCE(bs.landline, ''),
			COALESCE(bs.address_line_1, ''),
			COALESCE(bs.address_line_2, ''),
			COALESCE(bs.city, ''),
			COALESCE(bs.state, ''),
			COALESCE(bs.country, ''),
			COALESCE(bs.zip_code, '')
		FROM business_suppliers bs
		WHERE bs.business_id = $1
		  AND bs.id = $2::uuid
		  AND bs.deleted_at IS NULL
		LIMIT 1
	`, businessID, supplierID).Scan(&name, &email, &mobile, &alt, &land, &line1, &line2, &city, &state, &country, &zipCode)
	if err != nil {
		if err == pgx.ErrNoRows {
			return PurchaseOrderSupplierDetails{}, ErrPurchaseOrderNotFound
		}
		return PurchaseOrderSupplierDetails{}, fmt.Errorf("load purchase order supplier details: %w", err)
	}

	return PurchaseOrderSupplierDetails{
		Name:    name.String,
		Email:   email.String,
		Phone:   firstNonEmpty(mobile.String, alt.String, land.String),
		Address: joinAddressParts(line1.String, line2.String, city.String, state.String, country.String, zipCode.String),
	}, nil
}

func getPurchaseOrderBusinessDetails(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID string) (PurchaseOrderBusinessDetails, error) {
	var name, email, phone sql.NullString

	if err := querier.QueryRow(ctx, `
		SELECT
			COALESCE(name, ''),
			COALESCE(business_email, ''),
			COALESCE(business_phone, '')
		FROM businesses
		WHERE id = $1::uuid
		LIMIT 1
	`, businessID).Scan(&name, &email, &phone); err != nil {
		if err == pgx.ErrNoRows {
			return PurchaseOrderBusinessDetails{}, ErrPurchaseOrderNotFound
		}
		return PurchaseOrderBusinessDetails{}, fmt.Errorf("load purchase order business details: %w", err)
	}

	return PurchaseOrderBusinessDetails{
		Name:  name.String,
		Email: email.String,
		Phone: phone.String,
	}, nil
}

func getPurchaseOrderLocationDetails(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID, locationID string) (PurchaseOrderLocationDetails, error) {
	var (
		name    sql.NullString
		email   sql.NullString
		mobile  sql.NullString
		alt     sql.NullString
		address sql.NullString
		land    sql.NullString
		city    sql.NullString
		state   sql.NullString
		country sql.NullString
		zipCode sql.NullString
	)

	if err := querier.QueryRow(ctx, `
		SELECT
			COALESCE(loc.location_name, ''),
			COALESCE(loc.email, ''),
			COALESCE(loc.mobile, ''),
			COALESCE(loc.alternate_contact_number, ''),
			COALESCE(loc.exact_address, ''),
			COALESCE(loc.landmark, ''),
			COALESCE(loc.city, ''),
			COALESCE(loc.state, ''),
			COALESCE(loc.country, ''),
			COALESCE(loc.zip_code, '')
		FROM business_locations loc
		WHERE loc.business_id = $1
		  AND loc.id = $2::uuid
		LIMIT 1
	`, businessID, locationID).Scan(&name, &email, &mobile, &alt, &address, &land, &city, &state, &country, &zipCode); err != nil {
		if err == pgx.ErrNoRows {
			return PurchaseOrderLocationDetails{}, ErrPurchaseOrderNotFound
		}
		return PurchaseOrderLocationDetails{}, fmt.Errorf("load purchase order location details: %w", err)
	}

	return PurchaseOrderLocationDetails{
		Name:    name.String,
		Email:   email.String,
		Phone:   firstNonEmpty(mobile.String, alt.String),
		Address: joinAddressParts(address.String, land.String, city.String, state.String, country.String, zipCode.String),
	}, nil
}

func listPurchaseOrderItemsByOrderID(ctx context.Context, querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, businessID, purchaseOrderID string) ([]PurchaseOrderItem, error) {
	rows, err := querier.Query(ctx, `
		SELECT
			purchase_order_items.id::text,
			COALESCE(purchase_order_items.purchase_order_id::text, ''),
			purchase_order_items.business_id::text,
			purchase_order_items.product_id::text,
			purchase_order_items.product_name,
			purchase_order_items.sku,
			purchase_order_items.unit,
			purchase_order_items.order_quantity,
			purchase_order_items.unit_cost_before_discount,
			purchase_order_items.discount_percentage,
			purchase_order_items.discount_amount,
			purchase_order_items.unit_cost_before_tax,
			purchase_order_items.product_tax_rate,
			purchase_order_items.tax_amount,
			purchase_order_items.net_cost,
			purchase_order_items.selling_price,
			purchase_order_items.line_cost,
			COALESCE(purchase_order_items.manufacture_date::text, ''),
			COALESCE(purchase_order_items.expiry_date::text, ''),
			COALESCE(purchase_order_items.lot_number, ''),
			COALESCE(ib.quantity_available, 0),
			COALESCE(purchase_order_items.balance_quantity, purchase_order_items.order_quantity),
			purchase_order_items.received_quantity,
			purchase_order_items.items_received,
			purchase_order_items.received_status,
			purchase_order_items.sort_order,
			purchase_order_items.created_at::text,
			purchase_order_items.updated_at::text
		FROM purchase_order_items
		LEFT JOIN purchase_orders po
		  ON po.id = purchase_order_items.purchase_order_id
		 AND po.business_id = purchase_order_items.business_id
		 AND po.deleted_at IS NULL
		LEFT JOIN inventory_balances ib
		  ON ib.business_id = purchase_order_items.business_id
		 AND ib.product_id = purchase_order_items.product_id
		 AND ib.location_id = po.location_id
		WHERE purchase_order_items.business_id = $1
		  AND purchase_order_items.purchase_order_id = $2::uuid
		  AND purchase_order_items.deleted_at IS NULL
		ORDER BY purchase_order_items.sort_order ASC, purchase_order_items.created_at ASC
	`, businessID, purchaseOrderID)
	if err != nil {
		return nil, fmt.Errorf("load purchase order items: %w", err)
	}
	defer rows.Close()

	items := make([]PurchaseOrderItem, 0)
	for rows.Next() {
		item, err := scanPurchaseOrderItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purchase order items: %w", err)
	}

	return items, nil
}

func listPurchaseOrderLogsByOrderID(ctx context.Context, querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, businessID, purchaseOrderID string) ([]PurchaseOrderLog, error) {
	rows, err := querier.Query(ctx, `
		SELECT
			pol.id::text,
			pol.business_id::text,
			pol.purchase_order_id::text,
			pol.action,
			COALESCE(u.id::text, ''),
			COALESCE(u.full_name, 'System'),
			COALESCE(pol.note, ''),
			pol.action_date::text
		FROM purchase_order_logs pol
		LEFT JOIN users u ON u.id = pol.actioned_by
		WHERE pol.business_id = $1
		  AND pol.purchase_order_id = $2::uuid
		ORDER BY pol.action_date ASC
	`, businessID, purchaseOrderID)
	if err != nil {
		return nil, fmt.Errorf("load purchase order logs: %w", err)
	}
	defer rows.Close()

	activities := make([]PurchaseOrderLog, 0)
	for rows.Next() {
		var logEntry PurchaseOrderLog
		var actionedByID sql.NullString
		var actionedByName sql.NullString
		if err := rows.Scan(
			&logEntry.ID,
			&logEntry.BusinessID,
			&logEntry.PurchaseOrderID,
			&logEntry.Action,
			&actionedByID,
			&actionedByName,
			&logEntry.Note,
			&logEntry.ActionDate,
		); err != nil {
			return nil, fmt.Errorf("scan purchase order log: %w", err)
		}
		if actionedByID.Valid || actionedByName.Valid {
			logEntry.ActionedBy = &PurchaseOrderCreatedBy{
				ID:   actionedByID.String,
				Name: actionedByName.String,
			}
		}
		activities = append(activities, logEntry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purchase order logs: %w", err)
	}

	return activities, nil
}

func listPurchaseOrderAdditionalExpensesByOrderID(ctx context.Context, querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, businessID, purchaseOrderID string) ([]PurchaseOrderExtraCost, error) {
	rows, err := querier.Query(ctx, `
		SELECT
			expense_name,
			amount,
			sort_order
		FROM purchase_order_additional_expenses
		WHERE business_id = $1
		  AND purchase_order_id = $2::uuid
		  AND deleted_at IS NULL
		ORDER BY sort_order ASC, created_at ASC
	`, businessID, purchaseOrderID)
	if err != nil {
		return nil, fmt.Errorf("load purchase order additional expenses: %w", err)
	}
	defer rows.Close()

	expenses := make([]PurchaseOrderExtraCost, 0)
	for rows.Next() {
		var expense PurchaseOrderExtraCost
		if err := rows.Scan(&expense.Name, &expense.Amount, &expense.SortOrder); err != nil {
			return nil, fmt.Errorf("scan purchase order additional expense: %w", err)
		}
		expenses = append(expenses, expense)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purchase order additional expenses: %w", err)
	}

	return expenses, nil
}

func listPurchaseOrderApprovalActivitiesByOrderID(ctx context.Context, querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, businessID, purchaseOrderID string) ([]PurchaseOrderLog, error) {
	rows, err := querier.Query(ctx, `
		SELECT
			poa.id::text,
			poa.business_id::text,
			poa.purchase_order_id::text,
			CASE
				WHEN poa.approval_status = 'pending_approval' THEN 'approval_requested'
				ELSE poa.approval_status
			END AS action,
			COALESCE(u.id::text, ''),
			COALESCE(u.full_name, 'System'),
			COALESCE(
				NULLIF(
					TRIM(
						CONCAT(
							CASE
								WHEN cardinality(COALESCE(poa.reminder_channels, ARRAY[]::text[])) > 0 THEN
									'Reminder channels: ' || array_to_string(ARRAY(
										SELECT initcap(channel)
										FROM unnest(poa.reminder_channels) AS channel
									), ', ')
								ELSE ''
							END,
							CASE
								WHEN COALESCE(poa.reminder_message, '') <> '' THEN
									CASE
										WHEN cardinality(COALESCE(poa.reminder_channels, ARRAY[]::text[])) > 0 THEN E'\n'
										ELSE ''
									END || 'Message: ' || poa.reminder_message
								ELSE ''
							END
						)
					),
					''
				),
				'Approval reminder queued.'
			) AS note,
			COALESCE(poa.requested_at, poa.created_at)::text
		FROM purchase_order_approvals poa
		LEFT JOIN users u ON u.id = poa.requested_by
		WHERE poa.business_id = $1
		  AND poa.purchase_order_id = $2::uuid
		ORDER BY COALESCE(poa.requested_at, poa.created_at) ASC
	`, businessID, purchaseOrderID)
	if err != nil {
		return nil, fmt.Errorf("load purchase order approval activities: %w", err)
	}
	defer rows.Close()

	activities := make([]PurchaseOrderLog, 0)
	for rows.Next() {
		var entry PurchaseOrderLog
		var actionedByID sql.NullString
		var actionedByName sql.NullString
		if err := rows.Scan(
			&entry.ID,
			&entry.BusinessID,
			&entry.PurchaseOrderID,
			&entry.Action,
			&actionedByID,
			&actionedByName,
			&entry.Note,
			&entry.ActionDate,
		); err != nil {
			return nil, fmt.Errorf("scan purchase order approval activity: %w", err)
		}
		if actionedByID.Valid || actionedByName.Valid {
			entry.ActionedBy = &PurchaseOrderCreatedBy{
				ID:   actionedByID.String,
				Name: actionedByName.String,
			}
		}
		activities = append(activities, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purchase order approval activities: %w", err)
	}

	return activities, nil
}

func sortPurchaseOrderActivities(activities []PurchaseOrderLog) {
	sort.SliceStable(activities, func(i, j int) bool {
		left := parseTimestampForOrdering(activities[i].ActionDate)
		right := parseTimestampForOrdering(activities[j].ActionDate)
		if left.Equal(right) {
			return activities[i].ID < activities[j].ID
		}
		return left.Before(right)
	})
}

func parseTimestampForOrdering(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}

	candidates := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}

	for _, layout := range candidates {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}

	return time.Time{}
}

func scanPurchaseOrderItem(scanner interface {
	Scan(dest ...any) error
}) (PurchaseOrderItem, error) {
	var (
		item          PurchaseOrderItem
		purchaseOrder sql.NullString
		expiryDate    sql.NullString
		currentStock  sql.NullFloat64
		receivedQty   sql.NullFloat64
	)

	if err := scanner.Scan(
		&item.ID,
		&purchaseOrder,
		&item.BusinessID,
		&item.ProductID,
		&item.ProductName,
		&item.SKU,
		&item.Unit,
		&item.OrderQuantity,
		&item.UnitCostBeforeDiscount,
		&item.DiscountPercentage,
		&item.DiscountAmount,
		&item.UnitCostBeforeTax,
		&item.ProductTaxRate,
		&item.TaxAmount,
		&item.NetCost,
		&item.SellingPrice,
		&item.LineCost,
		&item.ManufactureDate,
		&expiryDate,
		&item.LotNumber,
		&currentStock,
		&item.BalanceQuantity,
		&receivedQty,
		&item.ItemsReceived,
		&item.ReceivedStatus,
		&item.SortOrder,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return PurchaseOrderItem{}, err
	}

	if purchaseOrder.Valid {
		item.PurchaseOrderID = &purchaseOrder.String
	}
	if expiryDate.Valid {
		item.ExpiryDate = expiryDate.String
	}
	if currentStock.Valid {
		item.CurrentStock = currentStock.Float64
	}
	if receivedQty.Valid {
		value := receivedQty.Float64
		item.ReceivedQuantity = &value
	}

	return item, nil
}

func joinAddressParts(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	return strings.Join(filtered, ", ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
